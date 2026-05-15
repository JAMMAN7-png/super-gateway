package main

import (
	"flag"
	"log"
	"os"

	"github.com/goozway/super-gateway/internal/config"
	"github.com/goozway/super-gateway/internal/router"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg := config.DefaultConfig()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Override port from env if set
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = 0 // will be parsed below — FIXME: just use env
	}

	// Build and start server
	server, err := router.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "0.0.0.0:3000"
	}
	if port := os.Getenv("PORT"); port != "" {
		addr = "0.0.0.0:" + port
	}

	log.Fatal(server.Listen(addr))
}
