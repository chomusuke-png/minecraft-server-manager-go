package main

import (
	"log"

	"minecraft-manager/internal/app"
	"minecraft-manager/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error cargando configuración global: %v", err)
	}

	app.Run(cfg)
}
