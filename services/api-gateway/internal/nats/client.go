package nats_client

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Connect establishes a connection to the NATS server.
// It takes the NATS server address and a logger.
func Connect(natsAddress string, logger *zap.Logger) (*nats.Conn, error) {
	logger.Info("Attempting to connect to NATS server", zap.String("address", natsAddress))

	// I should connect to NATS with retry logic.
	nc, err := nats.Connect(
		natsAddress,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second*2),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warn("NATS connection closed")
		}),
	)

	if err != nil {
		logger.Error("Failed to connect to NATS", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", natsAddress, err)
	}

	logger.Info("Successfully connected to NATS", zap.String("url", nc.ConnectedUrl()))
	return nc, nil
}

// // Optional: Function to connect to NATS JetStream for persistent messaging
// func ConnectJetStream(nc *nats.Conn, logger *zap.Logger) (nats.JetStreamContext, error) {
// 	logger.Info("Attempting to connect to NATS JetStream")
// 	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
// 	if err != nil {
// 		logger.Error("Failed to connect to NATS JetStream", zap.Error(err))
// 		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
// 	}
// 	logger.Info("Successfully connected to NATS JetStream")

// 	// Optional: Create a stream if it doesn't exist (example)
// 	// streamName := "JOBS"
// 	// _, err = js.AddStream(&nats.StreamConfig{
// 	// 	Name:     streamName,
// 	// 	Subjects: []string{"jobs.*"}, // Subject pattern for the stream
// 	// 	Storage:  nats.FileStorage, // Or MemoryStorage
// 	// })
// 	// if err != nil && err != nats.ErrStreamNameAlreadyInUse {
// 	// 	logger.Error("Failed to create JetStream stream", zap.String("stream", streamName), zap.Error(err))
// 	// 	return nil, fmt.Errorf("failed to create stream %s: %w", streamName, err)
// 	// } else if err == nil {
// 	// 	logger.Info("Created NATS JetStream stream", zap.String("stream", streamName))
// 	// }

// 	return js, nil
// }
