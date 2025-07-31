package handlers

import (
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// LogHandler holds dependencies for log-related handlers.
type LogHandler struct {
	Logger  *zap.Logger
	LokiURL string
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(logger *zap.Logger, lokiURL string) *LogHandler {
	return &LogHandler{
		Logger:  logger,
		LokiURL: lokiURL,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for now.
		// In production, you would want to check the origin.
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// StreamLogs handles the WebSocket connection for streaming logs from Loki.
func (h *LogHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
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