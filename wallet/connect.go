package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

// WalletClient represents the wallet side of WalletConnect
type WalletClient struct {
	bridge         string
	key            string
	clientId       string
	peerMeta       PeerMeta
	conn           *websocket.Conn
	handshakeTopic string
	connected      bool
}

// PeerMeta contains metadata about the connected dApp
type PeerMeta struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Icons       []string `json:"icons"`
	Name        string   `json:"name"`
}

// SessionRequest represents an incoming session request
type SessionRequest struct {
	PeerId   string   `json:"peerId"`
	PeerMeta PeerMeta `json:"peerMeta"`
	ChainId  int      `json:"chainId"`
}

// ParseWalletConnectURI parses a WalletConnect URI and returns connection details
func ParseWalletConnectURI(uri string) (bridge string, handshakeTopic string, key string, err error) {
	// Remove "wc:" prefix if present
	uri = strings.TrimPrefix(uri, "wc:")

	// Split the URI into base and query parts
	parts := strings.SplitN(uri, "?", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid URI format: missing query parameters")
	}

	// Parse the base part (topic@version)
	baseParts := strings.Split(parts[0], "@")
	if len(baseParts) != 2 {
		return "", "", "", fmt.Errorf("invalid URI format: missing version")
	}
	handshakeTopic = baseParts[0]
	key = baseParts[1] // This is the version number

	// Parse query parameters
	query := make(map[string]string)
	for _, param := range strings.Split(parts[1], "&") {
		if param == "" {
			continue
		}
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			query[kv[0]] = kv[1]
		} else {
			query[kv[0]] = ""
		}
	}

	// Get bridge from query parameters, use default if empty
	bridge = query["bridge"]
	if bridge == "" {
		bridge = "wss://bridge.walletconnect.org"
	}

	return bridge, handshakeTopic, key, nil
}

// ConnectToURI connects to a dApp using a WalletConnect URI
func ConnectToURI(uri string, walletAddress common.Address) (*WalletClient, error) {
	bridge, topic, key, err := ParseWalletConnectURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %v", err)
	}

	client := &WalletClient{
		bridge:         bridge,
		key:            key,
		handshakeTopic: topic,
	}

	// Connect to bridge
	conn, _, err := websocket.DefaultDialer.Dial(bridge, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bridge: %v", err)
	}
	client.conn = conn

	// Subscribe to session request topic
	if err := client.subscribe(topic); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	client.connected = true
	return client, nil
}

// HandleSessionRequest processes an incoming session request
func (c *WalletClient) HandleSessionRequest(ctx context.Context) (*SessionRequest, error) {
	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := c.conn.ReadJSON(&msg); err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	if msg.Type != "pub" {
		return nil, nil // Not a session request
	}

	var request SessionRequest
	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		return nil, fmt.Errorf("failed to parse session request: %v", err)
	}

	c.peerMeta = request.PeerMeta
	return &request, nil
}

// ApproveSession approves a session request and sends wallet info to dApp
func (c *WalletClient) ApproveSession(address common.Address, chainId int) error {
	payload := map[string]interface{}{
		"approved": true,
		"chainId":  chainId,
		"accounts": []string{address.Hex()},
		"peerId":   c.clientId,
		"peerMeta": map[string]interface{}{
			"name":        "omonOmon Wallet",
			"description": "Go Console-based Wallet Implementation",
			"icons":       []string{},
		},
	}

	msg := struct {
		Topic   string      `json:"topic"`
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Topic:   c.handshakeTopic,
		Type:    "pub",
		Payload: payload,
	}

	return c.conn.WriteJSON(msg)
}

// HandleRequests listens for and handles incoming requests from the dApp
func (c *WalletClient) HandleRequests(ctx context.Context, handlers RequestHandlers) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var msg struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}

			if err := c.conn.ReadJSON(&msg); err != nil {
				return fmt.Errorf("failed to read message: %v", err)
			}

			// Handle different request types
			switch msg.Type {
			case "eth_sendTransaction":
				if handlers.SendTransaction != nil {
					handlers.SendTransaction(msg.Payload)
				}
			case "eth_sign":
				if handlers.Sign != nil {
					handlers.Sign(msg.Payload)
				}
			case "personal_sign":
				if handlers.PersonalSign != nil {
					handlers.PersonalSign(msg.Payload)
				}
			}
		}
	}
}

// RequestHandlers contains callback functions for different request types
type RequestHandlers struct {
	SendTransaction func(json.RawMessage)
	Sign            func(json.RawMessage)
	PersonalSign    func(json.RawMessage)
}

func (c *WalletClient) subscribe(topic string) error {
	msg := struct {
		Topic string `json:"topic"`
		Type  string `json:"type"`
	}{
		Topic: topic,
		Type:  "sub",
	}
	return c.conn.WriteJSON(msg)
}

// Close closes the WalletConnect connection
func (c *WalletClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (w *Wallet) WalletConnect(uri string) error {
	address := common.HexToAddress(w.Address)

	client, err := ConnectToURI(uri, address)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to WalletConnect bridge")

	ctx := context.Background()

	// Handle session request
	request, err := client.HandleSessionRequest(ctx)
	if err != nil {
		return fmt.Errorf("failed to handle session request: %v", err)
	}

	// Display dApp info and ask for confirmation
	fmt.Printf("\nConnection request from dApp:\n")
	fmt.Printf("Name: %s\n", request.PeerMeta.Name)
	fmt.Printf("URL: %s\n", request.PeerMeta.URL)
	fmt.Printf("Description: %s\n", request.PeerMeta.Description)

	fmt.Print("\nApprove connection? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		return fmt.Errorf("connection rejected by user")
	}

	// Approve session
	if err := client.ApproveSession(address, request.ChainId); err != nil {
		return fmt.Errorf("failed to approve session: %v", err)
	}

	fmt.Println("Connection approved! Listening for requests...")

	// Set up request handlers
	handlers := RequestHandlers{
		SendTransaction: func(payload json.RawMessage) {
			fmt.Printf("\nTransaction request received: %s\n", string(payload))
			request := TransactionRequest{}
			if err := json.Unmarshal(payload, &request); err != nil {
				fmt.Printf("failed to parse transaction request: %v", err)
				return
			}

			txHash, err := w.SendTransactionFromRequest(ctx, request)
			if err != nil {
				fmt.Printf("failed to send transaction: %v", err)
				return
			}
			fmt.Printf("Transaction sent with hash: %s\n", txHash)
		},
		Sign: func(payload json.RawMessage) {
			fmt.Printf("\nSign request received: %s\n", string(payload))
			signature, err := w.Sign(payload)
			if err != nil {
				fmt.Printf("failed to sign message: %v", err)
				return
			}
			fmt.Printf("Signature: %s\n", signature)
		},
		PersonalSign: func(payload json.RawMessage) {
			fmt.Printf("\nPersonal sign request received: %s\n", string(payload))
			signature, err := w.PersonalSign(string(payload))
			if err != nil {
				fmt.Printf("failed to sign message: %v", err)
				return
			}
			fmt.Printf("Signature: %s\n", signature)
		},
	}

	// Handle incoming requests until interrupted
	return client.HandleRequests(ctx, handlers)
}
