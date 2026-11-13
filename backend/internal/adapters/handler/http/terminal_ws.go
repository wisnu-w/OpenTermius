package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/opentermius/opentermius/internal/ports"
	"github.com/opentermius/opentermius/internal/usecase"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Configurable via CORS middleware
	},
}

type TerminalWSHandler struct {
	terminalUC *usecase.TerminalUseCase
	sshService ports.SSHService
}

func NewTerminalWSHandler(terminalUC *usecase.TerminalUseCase, sshService ports.SSHService) *TerminalWSHandler {
	return &TerminalWSHandler{
		terminalUC: terminalUC,
		sshService: sshService,
	}
}

type terminalInitPayload struct {
	Type       string `json:"type"` // "init"
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthType   string `json:"authType"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	Passphrase string `json:"passphrase"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
}

type terminalResizePayload struct {
	Type string `json:"type"` // "resize"
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// safeWSConn wraps *websocket.Conn with a mutex to prevent concurrent write panics
type safeWSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeWSConn) WriteMessage(messageType int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(messageType, data)
}

func (s *safeWSConn) Close() error {
	return s.conn.Close()
}

type wsWriter struct {
	safeConn *safeWSConn
	session  *usecase.TerminalSession
}

func (w *wsWriter) Write(p []byte) (int, error) {
	// Broadcast to active multiplayer viewers
	if w.session != nil {
		w.session.Broadcast(p)
	}

	if err := w.safeConn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (h *TerminalWSHandler) HandleTerminal(w http.ResponseWriter, r *http.Request) {
	rawConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal websocket upgrade failed: %v", err)
		return
	}

	conn := &safeWSConn{conn: rawConn}
	defer conn.Close()

	hostIDStr := r.PathValue("hostId")
	hostID, _ := uuid.Parse(hostIDStr)
	uid := getUserID(r)

	// Create multiplayer session entry
	session := h.terminalUC.CreateSession(uuid.Nil, hostID, uid, "SSH Terminal")
	defer h.terminalUC.CloseSession(r.Context(), session.ID, "")

	// 1. Wait for initial connection handshake
	_, msgBytes, err := rawConn.ReadMessage()
	if err != nil {
		return
	}

	var initPayload terminalInitPayload
	if err := json.Unmarshal(msgBytes, &initPayload); err != nil || initPayload.Type != "init" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[1;31mError: Missing initial SSH connection payload.\x1b[0m\r\n"))
		return
	}

	if initPayload.Host == "" {
		initPayload.Host = "127.0.0.1"
	}
	if initPayload.Port <= 0 {
		initPayload.Port = 22
	}
	if initPayload.Username == "" {
		initPayload.Username = "root"
	}
	if initPayload.Cols <= 0 {
		initPayload.Cols = 80
	}
	if initPayload.Rows <= 0 {
		initPayload.Rows = 24
	}

	// Announce connection attempt in terminal
	startNotice := fmt.Sprintf("\r\n\x1b[1;33mConnecting to %s@%s:%d via SSH...\x1b[0m\r\n", initPayload.Username, initPayload.Host, initPayload.Port)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(startNotice))

	writer := &wsWriter{safeConn: conn, session: session}

	sshAuth := ports.SSHAuth{
		Type:       initPayload.AuthType,
		Password:   initPayload.Password,
		PrivateKey: initPayload.PrivateKey,
		Passphrase: initPayload.Passphrase,
	}

	sshSession, err := h.sshService.Connect(
		r.Context(),
		initPayload.Host,
		initPayload.Port,
		initPayload.Username,
		sshAuth,
		initPayload.Cols,
		initPayload.Rows,
		writer,
	)
	if err != nil {
		errNotice := fmt.Sprintf("\r\n\x1b[1;31mSSH connection failed: %v\x1b[0m\r\n", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(errNotice))
		return
	}
	defer sshSession.Close()

	// Notify client of successful connection
	connNotice := fmt.Sprintf("\x1b[1;32m● Connected to %s@%s:%d\x1b[0m\r\n\r\n", initPayload.Username, initPayload.Host, initPayload.Port)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(connNotice))

	// Read input from client and forward to SSH stdin or handle resize
	for {
		messageType, p, err := rawConn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage {
			// Check if message is a resize control command
			var resizeCmd terminalResizePayload
			if err := json.Unmarshal(p, &resizeCmd); err == nil && resizeCmd.Type == "resize" {
				_ = sshSession.Resize(resizeCmd.Cols, resizeCmd.Rows)
				continue
			}
		}

		// Forward raw interactive keystrokes to SSH PTY stdin
		if _, err := sshSession.Write(p); err != nil {
			break
		}
	}
}

func (h *TerminalWSHandler) JoinSession(w http.ResponseWriter, r *http.Request) {
	rawConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Join session websocket upgrade failed: %v", err)
		return
	}

	conn := &safeWSConn{conn: rawConn}
	defer conn.Close()

	sessionIDStr := r.PathValue("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid session id"}`))
		return
	}

	session := h.terminalUC.GetSession(sessionID)
	if session == nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"session not found or inactive"}`))
		return
	}

	listenerID := uuid.New()
	stream := session.Subscribe(listenerID)
	defer session.Unsubscribe(listenerID)

	// Stream broadcast messages to multiplayer viewer
	for msg := range stream {
		if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			break
		}
	}
}
