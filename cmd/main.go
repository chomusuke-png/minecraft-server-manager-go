package main

import (
	"fmt"
	"log"
	"minecraft-manager/internal/config"
	"minecraft-manager/internal/downloader"
)

func main() {
	fmt.Println("[*] Iniciando Minecraft Server Manager...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error crítico cargando configuración: %v", err)
	}

	fmt.Printf("[+] Configuración cargada. RAM: %d GB | Puerto: %d\n", cfg.RAMGB, cfg.Port)

	dl := downloader.New("server")

	if cfg.PlayitPath == "playit.exe" {
		dl.DownloadPlayit()
	}

	fmt.Println("\n[*] Verificando archivos del servidor...")
	if dl.PromptUser() {
		fmt.Println("[*] ¡Servidor listo para arrancar!")
	} else {
		fmt.Println("[!] Operación cancelada o fallida.")
	}
}
