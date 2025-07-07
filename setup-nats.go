package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	fmt.Println("=== SETTING UP NATS STREAMS FOR DANTEGPU ===")

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("Failed to create JetStream context:", err)
	}

	// Stream configurations
	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name:     "TASKS",
			subjects: []string{"tasks.>", "jobs.>", "dispatch.>"},
		},
		{
			name:     "BILLING",
			subjects: []string{"billing.>", "payments.>", "transactions.>"},
		},
		{
			name:     "MONITORING", 
			subjects: []string{"metrics.>", "health.>", "status.>"},
		},
		{
			name:     "GPU",
			subjects: []string{"gpu.>", "provider.>", "rental.>"},
		},
		{
			name:     "NOTIFICATIONS",
			subjects: []string{"notifications.>", "alerts.>", "events.>"},
		},
	}

	// Create streams
	for _, stream := range streams {
		fmt.Printf("Creating stream: %s with subjects: %v\n", stream.name, stream.subjects)
		
		cfg := &nats.StreamConfig{
			Name:      stream.name,
			Subjects:  stream.subjects,
			Storage:   nats.FileStorage,
			MaxMsgs:   1000000,
			MaxBytes:  100 * 1024 * 1024, // 100MB
			MaxAge:    24 * time.Hour,     // 24 hours
			Retention: nats.LimitsPolicy,
			Discard:   nats.DiscardOld,
			Replicas:  1,
		}

		info, err := js.AddStream(cfg)
		if err != nil {
			// Try to update if stream already exists
			info, err = js.UpdateStream(cfg)
			if err != nil {
				log.Printf("Failed to create/update stream %s: %v", stream.name, err)
				continue
			}
		}
		
		fmt.Printf("✅ Stream %s created successfully\n", info.Config.Name)
	}

	// Verify streams
	fmt.Println("\n=== VERIFYING STREAMS ===")
	for name := range js.StreamNames() {
		info, err := js.StreamInfo(name)
		if err != nil {
			log.Printf("Error getting info for stream %s: %v", name, err)
			continue
		}
		fmt.Printf("✅ Stream: %s, Subjects: %v, Messages: %d\n", 
			info.Config.Name, info.Config.Subjects, info.State.Msgs)
	}

	fmt.Println("\n=== NATS STREAMS SETUP COMPLETE ===")
} 