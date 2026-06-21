package consul_client

import (
	"fmt"
	"net"
	"strconv"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/config"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// Connect establishes a connection to the Consul agent.
func Connect(consulAddress string, logger *zap.Logger) (*consulapi.Client, error) {
	logger.Info("Attempting to connect to Consul agent", zap.String("address", consulAddress))
	clientConfig := consulapi.DefaultConfig()
	clientConfig.Address = consulAddress
	client, err := consulapi.NewClient(clientConfig)
	if err != nil {
		logger.Error("Failed to create Consul client", zap.Error(err))
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}
	_, err = client.Agent().Self() // Ping agent to ensure connectivity
	if err != nil {
		logger.Error("Failed to ping Consul agent", zap.Error(err))
		return nil, fmt.Errorf("failed to connect/ping consul agent: %w", err)
	}
	logger.Info("Successfully connected to Consul agent", zap.String("address", consulAddress))
	return client, nil
}

// RegisterService registers this service instance with Consul.
func RegisterService(consulClient *consulapi.Client, cfg *config.Config, serviceID string, logger *zap.Logger) error {
	host, portStr, err := net.SplitHostPort(cfg.Port)
	if err != nil {
		// If format is not host:port (e.g. just :8003 or 8003), assume port only
		portStr = cfg.Port
		if portStr[0] == ':' {
			portStr = portStr[1:]
		}
		host = "" // Let Consul determine the address
		logger.Debug("Port config does not include host, Consul will use agent default address", zap.String("port_config", cfg.Port))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Error("Invalid port number in config", zap.String("port_str", portStr), zap.Error(err))
		return fmt.Errorf("invalid port number '%s': %w", portStr, err)
	}

	address := host // If host was empty, Consul agent's address is used by default.

	registration := &consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    cfg.ServiceName,
		Port:    port,
		Address: address,
		Tags:    cfg.ServiceTags,
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", getCheckAddress(address, logger), port, cfg.HealthCheckPath),
			Interval:                       cfg.HealthCheckInterval.String(),
			Timeout:                        cfg.HealthCheckTimeout.String(),
			DeregisterCriticalServiceAfter: "1m", // Example: Deregister after 1 minute of being critical
			Notes:                          "Health check for Scheduler Orchestrator Service",
		},
	}

	logger.Info("Attempting to register service with Consul",
		zap.String("service_id", serviceID),
		zap.String("service_name", cfg.ServiceName),
		zap.String("address", address),
		zap.Int("port", port),
		zap.String("check_url", registration.Check.HTTP),
	)

	err = consulClient.Agent().ServiceRegister(registration)
	if err != nil {
		logger.Error("Failed to register service with Consul", zap.Error(err))
		return fmt.Errorf("failed to register service '%s' with Consul: %w", cfg.ServiceName, err)
	}

	return nil
}

// getCheckAddress determines the address to use for the Consul health check URL.
// If the provided service address is empty or unspecified (0.0.0.0 or ::), use 127.0.0.1.
func getCheckAddress(serviceAddress string, logger *zap.Logger) string {
	if serviceAddress == "" || serviceAddress == "0.0.0.0" || serviceAddress == "::" {
		logger.Debug("Service address for health check is unspecified, using 127.0.0.1")
		return "127.0.0.1"
	}
	return serviceAddress
}

// DeregisterService deregisters the service from Consul.
// Typically called during graceful shutdown.
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

// DiscoverService finds healthy instances of a service registered in Consul.
func DiscoverService(consulClient *consulapi.Client, serviceName string, logger *zap.Logger) ([]*consulapi.ServiceEntry, error) {
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
		// Returning an empty slice and no error is often preferred over an error for "not found"
		// unless it's critical that at least one instance exists.
		// For the scheduler needing the provider-registry, it might be an error.
		// Let's return an error for now, can be changed based on usage.
		return nil, fmt.Errorf("no healthy instances found for service '%s'", serviceName)
	}

	logger.Debug("Discovered healthy service instances",
		zap.String("service", serviceName),
		zap.Int("count", len(serviceEntries)),
	)
	return serviceEntries, nil
}
