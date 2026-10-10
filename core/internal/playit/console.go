package playit

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	ansiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	// formato de playit --stdout: <fecha> <nivel> <modulo>: <mensaje>
	playitLogLine = regexp.MustCompile(`^\S+\s+(TRACE|DEBUG|INFO|WARN|ERROR)\s+\S+?:\s(.*)$`)
	claimURL      = regexp.MustCompile(`https://playit\.gg/claim/\S+`)
	tunnelCount   = regexp.MustCompile(`(\d+) tunnels? registered`)
	digits        = regexp.MustCompile(`[0-9]+`)
)

// consoleFilter decide que lineas de playit llegan a la consola del server. Con
// --stdout playit escribe su estado cada pocos segundos y las sesiones UDP, y
// todo eso tapaba la consola de minecraft; el log sigue guardando todo
type consoleFilter struct {
	w       io.Writer
	pending []byte

	shownClaims map[string]bool
	shownIssues map[string]bool
	approved    bool
	tunnels     int
	// la linea anterior fue el intento fallido por IPv6, que playit sigue con un
	// error generico sin direccion
	afterIPv6Fallback bool
}

func newConsoleFilter(w io.Writer) *consoleFilter {
	return &consoleFilter{
		w:           w,
		shownClaims: map[string]bool{},
		shownIssues: map[string]bool{},
		tunnels:     -1,
	}
}

// Write junta la salida hasta tener lineas completas. Devuelve len(p) por lo
// mismo que ansiStripper: una escritura corta cortaria tambien el log
func (f *consoleFilter) Write(p []byte) (int, error) {
	f.pending = append(f.pending, p...)
	for {
		end := bytes.IndexByte(f.pending, '\n')
		if end < 0 {
			break
		}
		line := string(f.pending[:end])
		f.pending = f.pending[end+1:]

		if message := f.filter(line); message != "" {
			if _, err := io.WriteString(f.w, message+"\n"); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}

// filter devuelve lo que hay que mostrar por la linea, o "" para descartarla
func (f *consoleFilter) filter(rawLine string) string {
	line := strings.TrimRight(ansiSequence.ReplaceAllString(rawLine, ""), "\r")
	if strings.TrimSpace(line) == "" {
		return ""
	}

	match := playitLogLine.FindStringSubmatch(line)
	if match == nil {
		// fuera del formato de siempre, como un panic: mejor que se vea
		return "[*] Playit: " + line
	}
	level, message := match[1], match[2]

	afterIPv6Fallback := f.afterIPv6Fallback
	f.afterIPv6Fallback = false

	if url := claimURL.FindString(message); url != "" {
		// playit repite el link cada pocos segundos hasta que se usa
		if f.shownClaims[url] {
			return ""
		}
		f.shownClaims[url] = true
		return "[!] Playit: abre este link para vincular el agente a tu cuenta: " + url
	}

	if strings.Contains(strings.ToLower(message), "approved") {
		if f.approved {
			return ""
		}
		f.approved = true
		return "[+] Playit: agente vinculado a tu cuenta."
	}

	if count := tunnelCount.FindStringSubmatch(message); count != nil {
		tunnels, _ := strconv.Atoi(count[1])
		if tunnels == f.tunnels {
			return ""
		}
		f.tunnels = tunnels
		if tunnels == 0 {
			return "[!] Playit: conectado, pero tu cuenta no tiene túneles. Crea uno para Minecraft Java desde el panel de playit.gg."
		}
		return fmt.Sprintf("[+] Playit: conectado, %d túnel(es) activo(s).", tunnels)
	}

	if level != "WARN" && level != "ERROR" {
		return ""
	}

	// al elegir servidor playit prueba primero por IPv6, y en una red sin IPv6
	// ese intento falla siempre antes de caer a IPv4, que es lo normal
	if strings.Contains(message, "failed to send initial ping") &&
		strings.Contains(message, "NetworkUnreachable") && strings.Contains(message, "addr=[") {
		f.afterIPv6Fallback = true
		return ""
	}
	if afterIPv6Fallback && strings.Contains(message, "failed to ping tunnel server") {
		return ""
	}

	// en el primer contacto la sesion todavia no existe y playit se autentica
	// justo despues: es parte del arranque, no un corte
	if strings.Contains(message, "reason=SessionNotSetup") {
		return ""
	}

	// los numeros cambian entre repeticiones del mismo aviso: tiempos, ids
	key := level + digits.ReplaceAllString(message, "#")
	if f.shownIssues[key] {
		return ""
	}
	f.shownIssues[key] = true

	if level == "ERROR" {
		return "[-] Playit: " + message
	}
	return "[!] Playit: " + message
}
