package main

import (
	"fmt"
	"log"
	config "minecraft-manager/internal/config"
)

func main() {
	fmt.Println("[*] Iniciando Minecraft Server Manager...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[-] Error crítico cargando configuración: %v", err)
	}

	fmt.Printf("[+] Configuración cargada correctamente:\n")
	fmt.Printf("    -> Java Path: %s\n", cfg.JavaPath)
	fmt.Printf("    -> RAM: %d GB\n", cfg.RAMGB)
	fmt.Printf("    -> Puerto: %d\n", cfg.Port)
	fmt.Printf("    -> Backup Days: %d\n", cfg.BackupRetentionDays)

	fmt.Println("[*] Sistema listo para los siguientes módulos.")
}
