package consul_client

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// Connect establishes a connection to the Consul agent.
func Connect(consulAddress string, logger *zap.Logger) (*consulapi.Client, error) {
	logger.Info("Attempting to connect to Consul agent", zap.String("address", consulAddress))

	config := consulapi.DefaultConfig()
	config.Address = consulAddress

	client, err := consulapi.NewClient(config)
	if err != nil {
		logger.Error("Failed to create Consul client", zap.Error(err))
		return nil, fmt.Errorf("failed to create consul client for address %s: %w", consulAddress, err)
	}

	// I should ping the agent to ensure connectivity.
	_, err = client.Agent().Self()
	if err != nil {
		logger.Error("Failed to ping Consul agent", zap.Error(err))
		return nil, fmt.Errorf("failed to connect/ping consul agent at %s: %w", consulAddress, err)
	}

	logger.Info("Successfully connected to Consul agent", zap.String("address", consulAddress))
	return client, nil
}

// DiscoverService finds healthy instances of a service registered in Consul.
func DiscoverService(consulClient *consulapi.Client, serviceName string, logger *zap.Logger) ([]*consulapi.ServiceEntry, error) {
	// I need to query Consul's health endpoint for the service.
	// PassingOnly=true ensures I only get healthy instances.
	// Empty tag means I get all instances regardless of tags.
	serviceEntries, _, err := consulClient.Health().Service(serviceName, "", true, nil)
	if err != nil {
		logger.Error("Failed to query Consul for service",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to discover service '%s' in Consul: %w", serviceName, err)
	}

	if len(serviceEntries) == 0 {
		logger.Warn("No healthy instances found for service in Consul", zap.String("service", serviceName))
		return nil, fmt.Errorf("no healthy instances found for service '%s'", serviceName)
	}

	logger.Debug("Discovered healthy service instances",
		zap.String("service", serviceName),
		zap.Int("count", len(serviceEntries)),
	)
	return serviceEntries, nil
}
