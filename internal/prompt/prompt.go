package prompt

import (
	"bufio"
	"fmt"
	"strings"

	"minecraft-manager/internal/logx"
)

func Loop[T any](reader *bufio.Reader, promptText string, accept func(input string) (value T, ok bool, errMsg string)) (value T, readOK bool) {
	for {
		fmt.Print(promptText)
		raw, err := reader.ReadString('\n')
		input := strings.TrimSpace(raw)

		value, accepted, errMsg := accept(input)
		if accepted {
			return value, true
		}

		if err != nil {
			var zero T
			return zero, false
		}

		logx.Error("%s", errMsg)
	}
}

func YesNo(reader *bufio.Reader, question string) bool {
	value, ok := Loop(reader, fmt.Sprintf("%s (y/n): ", question), func(input string) (bool, bool, string) {
		switch strings.ToLower(input) {
		case "y", "s", "si", "yes":
			return true, true, ""
		case "n", "no":
			return false, true, ""
		}
		return false, false, "Entrada incorrecta, reintente."
	})
	if !ok {
		logx.Error("\nNo se pudo leer la respuesta, se asume 'no'.")
		return false
	}
	return value
}

func LoopDefault[T any](reader *bufio.Reader, promptText string, defaultValue T, parse func(input string) (value T, ok bool, errMsg string)) T {
	value, ok := Loop(reader, promptText, func(input string) (T, bool, string) {
		if input == "" {
			return defaultValue, true, ""
		}
		return parse(input)
	})
	if !ok {
		return defaultValue
	}
	return value
}
