package prompt

import (
	"bufio"
	"strings"
	"testing"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func TestLoopRetriesUntilAccepted(t *testing.T) {
	got, ok := Loop(readerFor("mal\ntambien mal\nbien\n"), "prompt: ", func(input string) (string, bool, string) {
		if input == "bien" {
			return input, true, ""
		}
		return "", false, "reintenta"
	})
	if !ok || got != "bien" {
		t.Errorf("got (%q, %v)", got, ok)
	}
}

func TestLoopFailsCleanlyOnEOF(t *testing.T) {
	// el reader se agota sin que ninguna linea sea aceptada: loop no debe
	// colgarse en un loop infinito, tiene que devolver ok=false
	got, ok := Loop(readerFor(""), "prompt: ", func(input string) (string, bool, string) {
		return "", false, "nunca se acepta"
	})
	if ok {
		t.Errorf("se esperaba ok=false, got (%q, %v)", got, ok)
	}
}

func TestLoopTrimsWhitespace(t *testing.T) {
	got, ok := Loop(readerFor("  con espacios  \n"), "prompt: ", func(input string) (string, bool, string) {
		return input, true, ""
	})
	if !ok || got != "con espacios" {
		t.Errorf("got %q", got)
	}
}

func TestYesNo(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"s\n", true},
		{"si\n", true},
		{"Y\n", true},
		{"n\n", false},
		{"no\n", false},
		{"N\n", false},
	}

	for _, c := range cases {
		if got := YesNo(readerFor(c.input), "¿Confirma?"); got != c.want {
			t.Errorf("YesNo(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestYesNoRetriesOnInvalidInput(t *testing.T) {
	if got := YesNo(readerFor("tal vez\ny\n"), "¿Confirma?"); got != true {
		t.Errorf("got %v", got)
	}
}

func TestYesNoAssumesNoWithoutInput(t *testing.T) {
	if got := YesNo(readerFor(""), "¿Confirma?"); got != false {
		t.Errorf("got %v, want false", got)
	}
}

func TestLoopDefaultUsesDefaultOnEmptyInput(t *testing.T) {
	got := LoopDefault(readerFor("\n"), "prompt: ", 42, func(input string) (int, bool, string) {
		t.Error("no debería llamar a parse con input vacío")
		return 0, false, ""
	})
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestLoopDefaultParsesNonEmptyInput(t *testing.T) {
	got := LoopDefault(readerFor("7\n"), "prompt: ", 42, func(input string) (int, bool, string) {
		if input == "7" {
			return 7, true, ""
		}
		return 0, false, "inválido"
	})
	if got != 7 {
		t.Errorf("got %d, want 7", got)
	}
}

func TestLoopDefaultFallsBackOnEOF(t *testing.T) {
	got := LoopDefault(readerFor(""), "prompt: ", 42, func(input string) (int, bool, string) {
		return 0, false, "nunca se acepta"
	})
	if got != 42 {
		t.Errorf("got %d, want 42 (default)", got)
	}
}

func TestDefault(t *testing.T) {
	if got := Default(OriginCurrent, 6); got != " [actual: 6]" {
		t.Errorf("got %q", got)
	}
	if got := Default(OriginDefault, "true"); got != " [por defecto: true]" {
		t.Errorf("got %q", got)
	}
}

func TestMenuQuestion(t *testing.T) {
	if got := MenuQuestion(3, EnterKeepsCurrent); got != "[?] Opción [1-3], Enter mantiene la actual: " {
		t.Errorf("con default = %q", got)
	}
	if got := MenuQuestion(8, ""); got != "[?] Opción [1-8]: " {
		t.Errorf("sin default = %q", got)
	}
}

func TestOptionsAlineaLasNotas(t *testing.T) {
	got := Options("  ", []string{"Playit", "ngrok", "Ninguno"}, []string{"recomendado", "actual"})
	want := "" +
		"  1) Playit   — recomendado\n" +
		"  2) ngrok    — actual\n" +
		"  3) Ninguno\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// las etiquetas con acentos se alinean por caracteres, no por bytes
func TestOptionsCuentaCaracteresNoBytes(t *testing.T) {
	got := Options("", []string{"Pacífico", "Difícil"}, []string{"a", "b"})
	want := "1) Pacífico   — a\n2) Difícil    — b\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
