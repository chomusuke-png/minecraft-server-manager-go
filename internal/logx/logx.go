// Package logx da un formato consistente a los mensajes de estado de la
// aplicación ([*]/[+]/[!]/[-]). No reemplaza los prompts ni menús
// interactivos: esos siguen usando fmt directamente.
package logx

import (
	"fmt"
	"strings"
)

func Info(format string, args ...any) {
	print("[*]", format, args...)
}

func Success(format string, args ...any) {
	print("[+]", format, args...)
}

func Warn(format string, args ...any) {
	print("[!]", format, args...)
}

func Error(format string, args ...any) {
	print("[-]", format, args...)
}

// Detail imprime una línea indentada, usada para sub-ítems de un mensaje
// previo (ej. cada mod deshabilitado o backup eliminado).
func Detail(format string, args ...any) {
	fmt.Printf("    -> "+format+"\n", args...)
}

func print(prefix, format string, args ...any) {
	if strings.HasPrefix(format, "\n") {
		fmt.Println()
		format = strings.TrimPrefix(format, "\n")
	}
	fmt.Printf(prefix+" "+format+"\n", args...)
}
