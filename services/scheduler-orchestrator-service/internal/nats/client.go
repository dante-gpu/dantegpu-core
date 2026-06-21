package nats_client

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Connect establishes a connection to the NATS server with specific options
// tailored for the scheduler-orchestrator service (e.g., robust reconnect logic).
func Connect(natsAddress string, logger *zap.Logger) (*nats.Conn, error) {
	logger.Info("Attempting to connect to NATS server for scheduler", zap.String("address", natsAddress))

	nc, err := nats.Connect(
		natsAddress,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(50),            // More aggressive reconnect attempts
		nats.ReconnectWait(time.Second*5), // Longer wait between reconnects
		nats.Timeout(10*time.Second),      // Connection timeout
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			} else {
				logger.Warn("NATS disconnected (no specific error)")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warn("NATS connection closed permanently. Will not attempt to reconnect.")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("NATS async error",
				zap.Stringp("subject", &sub.Subject),
				zap.Stringp("queue_group", &sub.Queue),
				zap.Error(err),
			)
		}),
	)

	if err != nil {
		logger.Error("Failed to connect to NATS after retries", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", natsAddress, err)
	}

	logger.Info("Successfully connected to NATS", zap.String("url", nc.ConnectedUrl()))
	return nc, nil
}

// ConnectJetStream establishes a JetStream context from a NATS connection.
// The scheduler will likely use JetStream for durable job queues.
func ConnectJetStream(nc *nats.Conn, logger *zap.Logger) (nats.JetStreamContext, error) {
	logger.Info("Attempting to get NATS JetStream context")
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256)) // Default, can be configured
	if err != nil {
		logger.Error("Failed to get NATS JetStream context", zap.Error(err))
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}
	logger.Info("Successfully obtained NATS JetStream context")
	return js, nil
}

// EnsureStream checks if a stream with the given configuration exists,
// and creates it if it doesn't.
func EnsureStream(js nats.JetStreamContext, streamName string, subjects []string, logger *zap.Logger) error {
	logger.Info("Ensuring NATS JetStream stream exists", zap.String("stream_name", streamName), zap.Strings("subjects", subjects))

	streamInfo, err := js.StreamInfo(streamName)
	// If stream doesn't exist, create it
	if err != nil && err == nats.ErrStreamNotFound {
		logger.Info("Stream not found, creating it...", zap.String("stream_name", streamName))
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
			Storage:  nats.FileStorage, // Or nats.MemoryStorage, make configurable
			// Other stream settings like retention, replicas can be added here
		})
		if err != nil {
			logger.Error("Failed to create NATS JetStream stream", zap.String("stream_name", streamName), zap.Error(err))
			return fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
		logger.Info("Successfully created NATS JetStream stream", zap.String("stream_name", streamName))
		return nil
	}
	if err != nil {
		// Other errors during StreamInfo check
		logger.Error("Failed to get NATS JetStream stream info", zap.String("stream_name", streamName), zap.Error(err))
		return fmt.Errorf("failed to get stream info for %s: %w", streamName, err)
	}

	// Stream exists, log info
	logger.Info("NATS JetStream stream already exists",
		zap.String("stream_name", streamInfo.Config.Name),
		zap.Uint64("messages", streamInfo.State.Msgs),
	)
	return nil
}
