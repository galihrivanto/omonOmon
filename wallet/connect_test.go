package wallet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func TestParseWalletConnectURI(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		wantBridge     string
		wantTopic      string
		wantKey        string
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:       "Valid URI with default bridge",
			uri:        "wc:8a5e5bdc-a0e4-4702-ba63-8f1a5655744f@1?bridge=&key=41791102999c339c844880b23950704cc43aa840f3739e365323cda4dfa89e7a",
			wantBridge: "wss://bridge.walletconnect.org",
			wantTopic:  "8a5e5bdc-a0e4-4702-ba63-8f1a5655744f",
			wantKey:    "1",
			wantErr:    false,
		},
		{
			name:       "Valid URI with custom bridge",
			uri:        "wc:8a5e5bdc-a0e4-4702-ba63-8f1a5655744f@1?bridge=wss://custom.bridge.org&key=41791102999c339c844880b23950704cc43aa840f3739e365323cda4dfa89e7a",
			wantBridge: "wss://custom.bridge.org",
			wantTopic:  "8a5e5bdc-a0e4-4702-ba63-8f1a5655744f",
			wantKey:    "1",
			wantErr:    false,
		},
		{
			name:           "Invalid URI format",
			uri:            "invalid-uri",
			wantErr:        true,
			expectedErrMsg: "invalid URI format",
		},
		{
			name:       "Test",
			uri:        "wc:3408fdf6bb9c288ccbb280aa4c91cc76e01ee9e2f7b4353439e970cb47a12c6e@2?expiryTimestamp=1740304195&relay-protocol=irn&symKey=2c696ec83a6f745f171e0af5de0a990370d9bec40e8fbda1a6540717119b57ed",
			wantBridge: "wss://bridge.walletconnect.org",
			wantTopic:  "3408fdf6bb9c288ccbb280aa4c91cc76e01ee9e2f7b4353439e970cb47a12c6e",
			wantKey:    "2",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge, topic, key, err := ParseWalletConnectURI(tt.uri)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantBridge, bridge)
			assert.Equal(t, tt.wantTopic, topic)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

func TestWalletClient_HandleSessionRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send test session request
		testRequest := struct {
			Type    string         `json:"type"`
			Payload SessionRequest `json:"payload"`
		}{
			Type: "pub",
			Payload: SessionRequest{
				PeerId: "test-peer",
				PeerMeta: PeerMeta{
					Description: "Test dApp",
					URL:         "https://test.com",
					Icons:       []string{"https://test.com/icon.png"},
					Name:        "Test",
				},
				ChainId: 1,
			},
		}
		conn.WriteJSON(testRequest)
	}))
	defer server.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := &WalletClient{
		bridge:    wsURL,
		key:       "test-key",
		clientId:  "test-client",
		connected: true,
	}

	// Connect to test server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to test server: %v", err)
	}
	client.conn = conn
	defer client.Close()

	// Test HandleSessionRequest
	request, err := client.HandleSessionRequest(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, "test-peer", request.PeerId)
	assert.Equal(t, "Test dApp", request.PeerMeta.Description)
	assert.Equal(t, 1, request.ChainId)
}

func TestWalletClient_ApproveSession(t *testing.T) {
	msgReceived := make(chan map[string]interface{})

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Receive approval message
		var receivedMsg map[string]interface{}
		err = conn.ReadJSON(&receivedMsg)
		if err != nil {
			t.Errorf("failed to read approval message: %v", err)
			close(msgReceived)
			return
		}

		// Send received message through channel
		msgReceived <- receivedMsg
	}))
	defer server.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := &WalletClient{
		bridge:         wsURL,
		key:            "test-key",
		clientId:       "test-client",
		handshakeTopic: "test-topic",
		connected:      true,
	}

	// Connect to test server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to test server: %v", err)
	}
	client.conn = conn
	defer client.Close()

	// Test ApproveSession
	address := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	err = client.ApproveSession(address, 1)
	assert.NoError(t, err)

	// Wait for server to receive message with timeout
	select {
	case receivedMsg := <-msgReceived:
		// Verify the approval message
		assert.Equal(t, "test-topic", receivedMsg["topic"])
		assert.Equal(t, "pub", receivedMsg["type"])

		payload, ok := receivedMsg["payload"].(map[string]interface{})
		assert.True(t, ok, "payload should be a map")
		if !ok {
			t.FailNow()
		}

		approved, ok := payload["approved"].(bool)
		assert.True(t, ok, "approved should be a boolean")
		assert.True(t, approved, "session should be approved")

		chainId, ok := payload["chainId"].(float64)
		assert.True(t, ok, "chainId should be a number")
		assert.Equal(t, float64(1), chainId)

		accounts, ok := payload["accounts"].([]interface{})
		assert.True(t, ok, "accounts should be an array")
		assert.Equal(t, []interface{}{address.Hex()}, accounts)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}
}

func TestWalletClient_HandleRequests(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send test requests
		testRequests := []struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}{
			{Type: "eth_sendTransaction", Payload: json.RawMessage(`{"to": "0x123"}`)},
			{Type: "eth_sign", Payload: json.RawMessage(`{"data": "0x456"}`)},
			{Type: "personal_sign", Payload: json.RawMessage(`{"message": "Hello"}`)},
		}

		for _, req := range testRequests {
			conn.WriteJSON(req)
		}
	}))
	defer server.Close()

	// Create client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := &WalletClient{
		bridge:    wsURL,
		key:       "test-key",
		clientId:  "test-client",
		connected: true,
	}

	// Connect to test server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to test server: %v", err)
	}
	client.conn = conn
	defer client.Close()

	// Create channels for synchronization
	sendTxCh := make(chan struct{})
	signCh := make(chan struct{})
	personalCh := make(chan struct{})

	handlers := RequestHandlers{
		SendTransaction: func(payload json.RawMessage) { close(sendTxCh) },
		Sign:            func(payload json.RawMessage) { close(signCh) },
		PersonalSign:    func(payload json.RawMessage) { close(personalCh) },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start handling requests in a goroutine
	go func() {
		client.HandleRequests(ctx, handlers)
	}()

	// Wait for all handlers to be called or timeout
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			t.Fatal("timeout waiting for handlers to be called")
		case <-sendTxCh:
			t.Log("SendTransaction handler called")
		case <-signCh:
			t.Log("Sign handler called")
		case <-personalCh:
			t.Log("PersonalSign handler called")
		}
	}
}
