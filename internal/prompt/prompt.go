package prompt

import (
	"bufio"
	"fmt"
	"strings"
	"unicode/utf8"

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

// convencion para los valores por defecto, igual en todo el programa:
//   - en una pregunta de texto el valor va entre corchetes con su origen,
//     ej. "[actual: 6]" o "[por defecto: 4]"
//   - en un menu la opcion se marca al final de su linea con "— actual" o
//     "— recomendado", y la pregunta dice que hace Enter
// los parentesis quedan para aclaraciones, como un rango

// origenes de un valor por defecto
const (
	OriginDefault = "por defecto"
	OriginCurrent = "actual"
	OriginNewest  = "más reciente"
)

// que hace Enter en un menu, para MenuQuestion
const (
	EnterKeepsCurrent     = "Enter mantiene la actual"
	EnterPicksRecommended = "Enter elige la recomendada"
	EnterPicksDefault     = "Enter elige la opción por defecto"
)

// Default arma el corchete de una pregunta de texto, ej. " [actual: 6]"
func Default(origin string, value any) string {
	return fmt.Sprintf(" [%s: %v]", origin, value)
}

// MenuQuestion arma la pregunta de un menu, ej. "[?] Opción [1-3], Enter
// mantiene la actual: ". enterHint vacio es un menu sin opcion por defecto
func MenuQuestion(options int, enterHint string) string {
	if enterHint == "" {
		return fmt.Sprintf("[?] Opción [1-%d]: ", options)
	}
	return fmt.Sprintf("[?] Opción [1-%d], %s: ", options, enterHint)
}

// Options arma un menu numerado con las notas alineadas en una columna, ej.
// "  1) Playit   — recomendado". notes puede ser mas corto que labels, y una
// nota vacia deja la linea sin marca
func Options(indent string, labels, notes []string) string {
	width := 0
	for i, label := range labels {
		if i < len(notes) && notes[i] != "" {
			width = max(width, utf8.RuneCountInString(label))
		}
	}

	var menu strings.Builder
	for i, label := range labels {
		note := ""
		if i < len(notes) {
			note = notes[i]
		}
		if note == "" {
			fmt.Fprintf(&menu, "%s%d) %s\n", indent, i+1, label)
			continue
		}
		padding := strings.Repeat(" ", width-utf8.RuneCountInString(label))
		fmt.Fprintf(&menu, "%s%d) %s%s   — %s\n", indent, i+1, label, padding, note)
	}
	return menu.String()
}
