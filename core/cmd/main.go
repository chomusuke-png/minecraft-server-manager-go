package main

import (
	"log"

	"minecraft-manager/internal/app"
	"minecraft-manager/internal/config"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error cargando configuración global: %v", err)
	}

	app.Run(cfg, version)
}
