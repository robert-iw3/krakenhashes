package heartbeat

import (
	"log"
	"time"
)

func Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sendHeartbeat()
	}
}

func sendHeartbeat() {
	// TODO: Implement actual heartbeat sending logic
	log.Println("Sending heartbeat...")
}
