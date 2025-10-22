package main

import (
	"context"
	"flag"
	"log"
	"time"

	"humidity-reminder/internal/app"
	"humidity-reminder/internal/config"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration YAML file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("missing required -config flag")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("run application: %v", err)
	}
}
