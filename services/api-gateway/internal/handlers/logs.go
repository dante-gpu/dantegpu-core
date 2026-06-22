package handlers

import (
	"net/http"
	"net/url"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/auth"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// LogHandler holds dependencies for log-related handlers.
type LogHandler struct {
	Logger         *zap.Logger
	LokiURL        string
	JwtSecret      string
	AllowedOrigins []string
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(logger *zap.Logger, lokiURL, jwtSecret string, allowedOrigins []string) *LogHandler {
	return &LogHandler{
		Logger:         logger,
		LokiURL:        lokiURL,
		JwtSecret:      jwtSecret,
		AllowedOrigins: allowedOrigins,
	}
}

// originAllowed permits same-origin requests (no Origin header, e.g. native
// clients) and any origin on the configured allowlist. This blocks cross-site
// WebSocket hijacking from arbitrary pages a logged-in operator might visit.
func (h *LogHandler) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range h.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	h.Logger.Warn("Rejected log stream from disallowed origin", zap.String("origin", origin))
	return false
}

// StreamLogs handles the WebSocket connection for streaming logs from Loki.
// NOTE: the Loki tail is still cluster-wide (container=~`.+`); restricting the
// stream to the requesting tenant and gating full-fleet access behind an admin
// role is a follow-up hardening step. Auth + origin checks below close the
// unauthenticated / any-origin exposure.
func (h *LogHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	// Browsers cannot attach an Authorization header to a WebSocket handshake, so
	// the console passes the JWT as a `token` query param. Validate it BEFORE the
	// upgrade so an unauthenticated caller never reaches the log stream.
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if _, err := auth.ValidateJWT(token, h.JwtSecret); err != nil {
		h.Logger.Warn("Rejected log stream: invalid token", zap.Error(err))
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:     h.originAllowed,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Upgrade the HTTP connection to a WebSocket connection.
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("Failed to upgrade connection to WebSocket", zap.Error(err))
		return
	}
	defer clientConn.Close()

	h.Logger.Info("Client connected for log streaming")

	// Parse the base Loki URL from the handler's configuration.
	parsedLokiURL, err := url.Parse(h.LokiURL)
	if err != nil {
		h.Logger.Error("Failed to parse Loki URL from config", zap.String("url", h.LokiURL), zap.Error(err))
		clientConn.WriteMessage(websocket.TextMessage, []byte("Error: Invalid log backend configuration."))
		return
	}

	// Define the Loki WebSocket URL.
	lokiRequestURL := url.URL{
		Scheme:   "ws", // Always ws for WebSocket
		Host:     parsedLokiURL.Host,
		Path:     "/loki/api/v1/tail",
		RawQuery: "query={container=~`.+`}",
	}

	h.Logger.Info("Connecting to Loki for log streaming", zap.String("url", lokiRequestURL.String()))

	// Connect to the Loki WebSocket endpoint.
	lokiConn, _, err := websocket.DefaultDialer.Dial(lokiRequestURL.String(), nil)
	if err != nil {
		h.Logger.Error("Failed to connect to Loki WebSocket", zap.Error(err))
		clientConn.WriteMessage(websocket.TextMessage, []byte("Error: Could not connect to log backend."))
		return
	}
	defer lokiConn.Close()

	h.Logger.Info("Successfully connected to Loki WebSocket")

	// Goroutine to read from client and close Loki connection if client disconnects.
	go func() {
		defer lokiConn.Close()
		defer clientConn.Close()
		for {
			// We don't expect messages from the client, but we read to detect closure.
			if _, _, err := clientConn.ReadMessage(); err != nil {
				h.Logger.Info("Client disconnected", zap.Error(err))
				return
			}
		}
	}()

	// Read messages from Loki and forward them to the client.
	for {
		messageType, p, err := lokiConn.ReadMessage()
		if err != nil {
			h.Logger.Error("Error reading message from Loki", zap.Error(err))
			// Inform the client that the connection to the backend was lost.
			clientConn.WriteMessage(websocket.TextMessage, []byte("Error: Connection to log backend lost."))
			return
		}

		// Forward the message to our client.
		if err := clientConn.WriteMessage(messageType, p); err != nil {
			h.Logger.Error("Error writing message to client", zap.Error(err))
			// Client connection is likely closed, so we exit.
			return
		}
	}
} 