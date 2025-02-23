package wallet

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransactionRequest represents a transaction request from WalletConnect
type TransactionRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Data     string `json:"data"`
	Value    string `json:"value"`
	GasPrice string `json:"gasPrice"`
	GasLimit string `json:"gas"`
	Nonce    string `json:"nonce"`
}

// SignRequest represents a signing request from WalletConnect
type SignRequest struct {
	Address string `json:"address"`
	Message string `json:"message"`
}

// TypedDataDomain represents the domain separator in EIP-712
type TypedDataDomain struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ChainId           *big.Int `json:"chainId"`
	VerifyingContract string   `json:"verifyingContract"`
	Salt              string   `json:"salt"`
}

// TypedDataType represents a type definition in EIP-712
type TypedDataType struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TypedData represents the complete typed data structure for EIP-712
type TypedData struct {
	Types       map[string][]TypedDataType `json:"types"`
	PrimaryType string                     `json:"primaryType"`
	Domain      TypedDataDomain            `json:"domain"`
	Message     map[string]interface{}     `json:"message"`
}

// SendTransactionFromRequest processes a WalletConnect transaction request
func (w *Wallet) SendTransactionFromRequest(ctx context.Context, req TransactionRequest) (string, error) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(w.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	// Verify the from address matches
	if !strings.EqualFold(req.From, fromAddress.Hex()) {
		return "", fmt.Errorf("from address mismatch")
	}

	// Parse transaction parameters
	to := common.HexToAddress(req.To)
	value := new(big.Int)
	if req.Value != "" {
		value.SetString(strings.TrimPrefix(req.Value, "0x"), 16)
	}

	var nonce uint64
	if req.Nonce != "" {
		nonce = hexutil.MustDecodeUint64(req.Nonce)
	} else {
		nonce, err = client.PendingNonceAt(ctx, fromAddress)
		if err != nil {
			return "", fmt.Errorf("failed to get nonce: %v", err)
		}
	}

	gasLimit := uint64(21000) // default
	if req.GasLimit != "" {
		gasLimit = hexutil.MustDecodeUint64(req.GasLimit)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %v", err)
	}

	if req.GasPrice != "" {
		gasPrice = new(big.Int)
		gasPrice.SetString(strings.TrimPrefix(req.GasPrice, "0x"), 16)
	}

	var data []byte
	if req.Data != "" {
		data, err = hex.DecodeString(strings.TrimPrefix(req.Data, "0x"))
		if err != nil {
			return "", fmt.Errorf("invalid data: %v", err)
		}
	}

	// Create and sign transaction
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	chainID := big.NewInt(CHAIN_ID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

// Sign implements eth_sign
func (w *Wallet) Sign(data []byte) ([]byte, error) {
	privateKey, err := crypto.HexToECDSA(w.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	// Add Ethereum prefix
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	hash := crypto.Keccak256Hash([]byte(msg))

	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %v", err)
	}

	// Convert signature to Ethereum format
	signature[64] += 27

	return signature, nil
}

// PersonalSign implements personal_sign
func (w *Wallet) PersonalSign(message string) (string, error) {
	// Decode message if it's hex encoded
	var data []byte
	if strings.HasPrefix(message, "0x") {
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(message, "0x"))
		if err != nil {
			data = []byte(message)
		}
	} else {
		data = []byte(message)
	}

	signature, err := w.Sign(data)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(signature), nil
}

// encodeType encodes a type definition
func encodeType(typeName string, types map[string][]TypedDataType) string {
	return encodeTypeWithContext(typeName, types, make(map[string]bool))
}

// encodeTypeWithContext encodes a type definition with context
func encodeTypeWithContext(typeName string, types map[string][]TypedDataType, processing map[string]bool) string {
	// Check for circular dependency
	if processing[typeName] {
		return "" // Skip if we're already processing this type
	}

	// Mark this type as being processed
	processing[typeName] = true
	defer delete(processing, typeName) // Clean up after we're done

	// Get the primary type
	primaryType := types[typeName]
	deps := findTypeDependencies(typeName, types)

	// Sort dependencies alphabetically for consistent encoding
	sort.Strings(deps)

	// Build encoded dependencies string
	var encoded []string
	for _, dep := range deps {
		if dep != typeName {
			result := encodeTypeWithContext(dep, types, processing)
			if result != "" {
				encoded = append(encoded, result)
			}
		}
	}

	// Encode the current type
	result := typeName + "("
	for i, field := range primaryType {
		if i > 0 {
			result += ","
		}
		result += field.Type + " " + field.Name
	}
	result += ")"

	// Combine all encoded types
	encoded = append(encoded, result)
	return strings.Join(encoded, "")
}

// findTypeDependencies finds all types that a type depends on
func findTypeDependencies(typeName string, types map[string][]TypedDataType) []string {
	deps := make(map[string]bool)
	queue := []string{typeName}

	for i := 0; i < len(queue); i++ {
		currentType := queue[i]
		fields := types[currentType]

		for _, field := range fields {
			// Check if field type is a custom type
			if _, exists := types[field.Type]; exists {
				if !deps[field.Type] {
					deps[field.Type] = true
					queue = append(queue, field.Type)
				}
			}
		}
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}
	return result
}

// TypedDataSign implements eth_signTypedData
func (w *Wallet) TypedDataSign(data TypedData) (string, error) {
	privateKey, err := crypto.HexToECDSA(w.PrivateKey)
	if err != nil {
	}

	// 1. Hash the type information
	// This ensures the structure of the data is part of what's signed
	domainType := encodeType("EIP712Domain", data.Types)
	domainHash := crypto.Keccak256([]byte(domainType))

	primaryType := encodeType(data.PrimaryType, data.Types)
	typeHash := crypto.Keccak256([]byte(primaryType))

	// 2. Create and hash the domain separator
	// This includes chain ID, contract address, etc. to prevent replay attacks
	domainValues, err := encodeDomainValues(data.Domain)
	if err != nil {
		return "", err
	}
	domainSeparator := crypto.Keccak256(append(domainHash, domainValues...))

	// 3. Hash the message data
	// This includes the actual data being signed
	messageValues, err := encodeMessageValues(data.Message, data.Types[data.PrimaryType])
	if err != nil {
		return "", err
	}
	messageHash := crypto.Keccak256(append(typeHash, messageValues...))

	// 4. Combine everything into the final hash
	// The "\x19\x01" prefix is specified by EIP-712
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", domainSeparator, messageHash))
	hash := crypto.Keccak256(rawData)

	// Sign the final hash
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign typed data: %v", err)
	}

	// Convert signature to Ethereum format
	signature[64] += 27

	return hexutil.Encode(signature), nil
}

// Helper functions for encoding values
func encodeDomainValues(domain TypedDataDomain) ([]byte, error) {
	// Implementation depends on your specific needs
	// This is a simplified version
	var values []byte
	values = append(values, []byte(domain.Name)...)
	values = append(values, []byte(domain.Version)...)
	if domain.ChainId != nil {
		values = append(values, domain.ChainId.Bytes()...)
	}
	if domain.VerifyingContract != "" {
		values = append(values, common.HexToAddress(domain.VerifyingContract).Bytes()...)
	}
	if domain.Salt != "" {
		salt, err := hex.DecodeString(strings.TrimPrefix(domain.Salt, "0x"))
		if err != nil {
			return nil, err
		}
		values = append(values, salt...)
	}
	return values, nil
}

func encodeMessageValues(message map[string]interface{}, types []TypedDataType) ([]byte, error) {
	// Implementation depends on your specific needs
	// This is a simplified version
	var values []byte
	for _, field := range types {
		val, ok := message[field.Name]
		if !ok {
			return nil, fmt.Errorf("missing value for field: %s", field.Name)
		}
		encoded, err := encodeValue(val, field.Type)
		if err != nil {
			return nil, err
		}
		values = append(values, encoded...)
	}
	return values, nil
}

func encodeValue(value interface{}, typ string) ([]byte, error) {
	// Implementation depends on your specific needs
	// This is a simplified version
	switch typ {
	case "string":
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string value")
		}
		return []byte(str), nil
	case "uint256":
		num, ok := value.(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid uint256 value")
		}
		return num.Bytes(), nil
	case "address":
		addr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid address value")
		}
		return common.HexToAddress(addr).Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}
