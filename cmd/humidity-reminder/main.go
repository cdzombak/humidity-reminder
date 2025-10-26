package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"humidity-reminder/internal/app"
	"humidity-reminder/internal/config"
)

var version = "<dev>"

func main() {
	configPath := flag.String("config", "", "Path to configuration YAML file")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

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
