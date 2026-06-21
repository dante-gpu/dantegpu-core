package consul_client

import (
	"fmt"
	"net"
	"strconv"

	// Use imports relative to this service's module path
	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/config"

	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// Connect establishes a connection to the Consul agent.
// (We already defined this in the api-gateway, should be moved to shared lib later)
func Connect(consulAddress string, logger *zap.Logger) (*consulapi.Client, error) {
	logger.Info("Attempting to connect to Consul agent", zap.String("address", consulAddress))
	config := consulapi.DefaultConfig()
	config.Address = consulAddress
	client, err := consulapi.NewClient(config)
	if err != nil {
		logger.Error("Failed to create Consul client", zap.Error(err))
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}
	_, err = client.Agent().Self() // Ping agent
	if err != nil {
		logger.Error("Failed to ping Consul agent", zap.Error(err))
		return nil, fmt.Errorf("failed to connect/ping consul agent: %w", err)
	}
	logger.Info("Successfully connected to Consul agent", zap.String("address", consulAddress))
	return client, nil
}

// RegisterService registers this service instance with Consul.
func RegisterService(consulClient *consulapi.Client, cfg *config.Config, serviceID string, logger *zap.Logger) error {
	// I need to get the service port.
	// Assumes cfg.Port is in format ":8002"
	host, portStr, err := net.SplitHostPort(cfg.Port)
	if err != nil {
		// If it doesn't contain ":", assume it's just the port number
		portStr = cfg.Port
		host = "" // Let Consul figure out the address
		logger.Warn("Could not split host/port, assuming config value is port", zap.String("port_config", cfg.Port))
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Error("Invalid port number in config", zap.String("port_config", cfg.Port), zap.Error(err))
		return fmt.Errorf("invalid port number '%s': %w", portStr, err)
	}

	// If host wasn't specified in the port string, Consul will use the agent's address.
	// We could try to determine the local IP here if needed, but often letting Consul handle it is fine.
	address := host

	// I need to define the service registration details.
	registration := &consulapi.AgentServiceRegistration{
		ID:      serviceID,       // Unique ID for this instance
		Name:    cfg.ServiceName, // Name of the service group
		Port:    port,            // Port the service listens on
		Address: address,         // Address the service listens on (optional, Consul defaults to agent IP)
		Tags:    cfg.ServiceTags, // Tags for filtering/discovery

		// I should configure the health check.
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", getCheckAddress(address), port, cfg.HealthCheckPath),
			Interval:                       cfg.HealthCheckInterval.String(),
			Timeout:                        cfg.HealthCheckTimeout.String(),
			DeregisterCriticalServiceAfter: "1m", // Example: Deregister after 1 minute of being critical
		},
	}

	// I need to register the service with the local Consul agent.
	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		logger.Error("Failed to register service with Consul", zap.Error(err))
		return fmt.Errorf("failed to register service '%s' with Consul: %w", cfg.ServiceName, err)
	}

	return nil
}

// getCheckAddress determines the address to use for the Consul health check URL.
// If the provided service address is empty or unspecified (0.0.0.0), use 127.0.0.1.
func getCheckAddress(serviceAddress string) string {
	if serviceAddress == "" || serviceAddress == "0.0.0.0" {
		return "127.0.0.1" // Health check often needs localhost
	}
	return serviceAddress
}

// DeregisterService is called during graceful shutdown.
// (This logic is usually called directly from main using client.Agent().ServiceDeregister())
/*
func DeregisterService(consulClient *consulapi.Client, serviceID string, logger *zap.Logger) error {
    logger.Info("Deregistering service from Consul", zap.String("service_id", serviceID))
    err := consulClient.Agent().ServiceDeregister(serviceID)
    if err != nil {
        logger.Error("Failed to deregister service from Consul", zap.String("service_id", serviceID), zap.Error(err))
        return fmt.Errorf("failed to deregister service '%s': %w", serviceID, err)
    }
    logger.Info("Successfully deregistered service from Consul", zap.String("service_id", serviceID))
    return nil
}
*/
