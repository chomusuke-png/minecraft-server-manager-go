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
