package loadbalancer

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	consulapi "github.com/hashicorp/consul/api"
)

// LoadBalancer defines the interface for selecting a backend server.
type LoadBalancer interface {
	// Next takes a list of available service entries and returns the URL of the next one to use.
	Next(services []*consulapi.ServiceEntry) (*url.URL, error)
}

// RoundRobin is a simple round-robin load balancer.
// It keeps track of the index of the last used server.
type RoundRobin struct {
	current uint64 // Atomically updated index
}

// Constants for metadata/tag checking
const (
	MetaKeyProtocol = "protocol" // Example key in Service Meta
	TagHTTPS        = "https"    // Example tag indicating HTTPS
	DefaultScheme   = "http"
)

// NewRoundRobin creates a new RoundRobin load balancer.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{current: 0}
}

// Next implements the LoadBalancer interface for RoundRobin.
// It now attempts to determine the scheme (http/https) from Consul data.
func (rr *RoundRobin) Next(services []*consulapi.ServiceEntry) (*url.URL, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("no available services for load balancing")
	}

	// I need to get the next index atomically.
	idx := atomic.AddUint64(&rr.current, 1)
	// Modulo operation to wrap around the list of services.
	selected := services[idx%uint64(len(services))]

	// I need to construct the URL for the selected service.
	address := selected.Service.Address
	if address == "" {
		address = selected.Node.Address // Fallback to node address
	}
	port := selected.Service.Port

	// --- Determine Scheme (http/https) ---
	// I will check Service Meta first, then Tags.
	scheme := DefaultScheme // Default to http

	// Check Meta for a "protocol" key.
	if protoMeta, ok := selected.Service.Meta[MetaKeyProtocol]; ok {
		protoMetaLower := strings.ToLower(protoMeta)
		if protoMetaLower == "https" || protoMetaLower == "http" {
			scheme = protoMetaLower
		} else {
			// Log a warning? Invalid protocol specified in meta.
			fmt.Printf("Warning: Invalid protocol '%s' specified in service meta for %s. Defaulting to %s.\n",
				protoMeta, selected.Service.ID, DefaultScheme)
		}
	} else {
		// If not in Meta, check Tags for an "https" tag.
		for _, tag := range selected.Service.Tags {
			if strings.ToLower(tag) == TagHTTPS {
				scheme = "https"
				break // Found https tag, no need to check further tags
			}
		}
	}
	// --- End Scheme Determination ---

	targetURL := fmt.Sprintf("%s://%s:%d", scheme, address, port)
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		// Maybe log this error with more context?
		return nil, fmt.Errorf("failed to parse target URL '%s' for service %s: %w",
			targetURL, selected.Service.ID, err)
	}

	return parsedURL, nil
}

// // Future: Implement other load balancing strategies
// type Random struct {}
// type LeastConnections struct {}
