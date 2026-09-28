package playit

import (
	"bytes"
	"io"
	"testing"
)

// una linea real de playit --stdout, con sus colores
const playitLine = "\x1b[2m2026-09-27T06:34:57.085877Z\x1b[0m \x1b[32m INFO\x1b[0m \x1b[2mplayit_cli::ui\x1b[0m\x1b[2m:\x1b[0m Visit link to setup https://playit.gg/claim/0faa22a6d3\n"

const playitLineClean = "2026-09-27T06:34:57.085877Z  INFO playit_cli::ui: Visit link to setup https://playit.gg/claim/0faa22a6d3\n"

func TestAnsiStripperLimpiaUnaLineaDePlayit(t *testing.T) {
	var out bytes.Buffer
	stripper := &ansiStripper{w: &out}

	if _, err := stripper.Write([]byte(playitLine)); err != nil {
		t.Fatal(err)
	}
	if out.String() != playitLineClean {
		t.Errorf("got %q\nwant %q", out.String(), playitLineClean)
	}
}

func TestAnsiStripperSecuenciaPartidaEntreEscrituras(t *testing.T) {
	var out bytes.Buffer
	stripper := &ansiStripper{w: &out}

	// cortado a mitad de cada secuencia posible: despues de ESC, despues de ESC[
	// y en medio de los parametros
	for _, chunk := range []string{"antes \x1b", "[3", "2mverde\x1b[", "0m despues"} {
		if _, err := stripper.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if out.String() != "antes verde despues" {
		t.Errorf("got %q", out.String())
	}
}

func TestAnsiStripperDejaPasarTextoSinColores(t *testing.T) {
	var out bytes.Buffer
	stripper := &ansiStripper{w: &out}

	text := "sin colores [no es una secuencia] 100%\n"
	if _, err := stripper.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if out.String() != text {
		t.Errorf("got %q", out.String())
	}
}

// si informara menos bytes de los recibidos, io.MultiWriter cortaria la copia
// a todos los destinos, consola incluida
func TestAnsiStripperNoCortaAlMultiWriter(t *testing.T) {
	var console, log bytes.Buffer
	writer := io.MultiWriter(&console, &ansiStripper{w: &log})

	n, err := writer.Write([]byte(playitLine))
	if err != nil {
		t.Fatalf("MultiWriter falló: %v", err)
	}
	if n != len(playitLine) {
		t.Errorf("n = %d, want %d", n, len(playitLine))
	}
	if console.String() != playitLine {
		t.Error("la consola tiene que recibir la línea original, con colores")
	}
	if log.String() != playitLineClean {
		t.Errorf("el log tiene que quedar limpio: %q", log.String())
	}
}
