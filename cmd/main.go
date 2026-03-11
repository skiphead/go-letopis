// main.go
package main

import (
	"log"

	"github.com/skiphead/go-letopis/internal/infra/config"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Telegram API: %s", cfg.Telegram)
	log.Printf("DB: %s@%s", cfg.DBConfig.User, cfg.DBConfig.Nodes[0])

}
