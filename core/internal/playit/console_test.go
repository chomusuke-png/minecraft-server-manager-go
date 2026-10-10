package playit

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// lineas con el formato real de playit --stdout. Las direcciones son inventadas
const (
	lineAutoRun   = "2026-09-28T01:58:50.000000Z  INFO playit_cli::ui: no command provided, doing auto run\n"
	lineSecret    = "2026-09-28T01:58:51.000000Z  INFO playit_cli::playit_secret: loading secret file_path=/home/x/.config/playit_gg/playit.toml\n"
	lineClaim     = "2026-09-28T01:58:52.000000Z  INFO playit_cli::ui: Visit link to setup https://playit.gg/claim/f0895b07ae\n"
	lineApprove   = "2026-09-28T01:58:53.000000Z  INFO playit_cli::ui: Approve program at https://playit.gg/claim/f0895b07ae\n"
	lineApproved  = "2026-09-28T01:58:54.000000Z  INFO playit_cli::ui: Program approved :). Secret code being setup.\n"
	lineIPv6Fail  = "2026-09-28T01:58:55.000000Z ERROR playit_agent_core::agent_control::address_selector: failed to send initial ping error=Os { code: 101, kind: NetworkUnreachable, message: \"Network unreachable\" } addr=[2001:db8::1]:5525\n"
	linePingFail  = "2026-09-28T01:58:55.000100Z ERROR playit_agent_core::agent_control::address_selector: failed to ping tunnel server\n"
	linePong      = "2026-09-28T01:58:55.100000Z  INFO playit_agent_core::agent_control::address_selector: got initial pong from tunnel server pong=Pong { client_addr: 203.0.113.7:34016, tunnel_addr: 198.51.100.1:5525 }\n"
	lineNotSetup  = "2026-09-28T01:58:56.000000Z  WARN playit_agent_core::agent_control::maintained_control: session expired reason=SessionNotSetup\n"
	lineUDPAuth   = "2026-09-28T01:59:09.000000Z  INFO playit_agent_core::playit_agent: udp channel requires auth, sent auth request\n"
	lineUDPReady  = "2026-09-28T01:59:09.100000Z  INFO playit_agent_core::playit_agent: udp session details received\n"
	lineNoTunnels = "2026-09-28T01:59:00.000000Z  INFO playit_cli::ui: playit (v0.17.1): 1790560740430 tunnel running, 0 tunnels registered\n"
	lineOneTunnel = "2026-09-28T02:10:00.000000Z  INFO playit_cli::ui: playit (v0.17.1): 1790561400000 tunnel running, 1 tunnels registered\n"
)

func filterAll(t *testing.T, lines ...string) string {
	t.Helper()
	var console bytes.Buffer
	filter := newConsoleFilter(&console)
	for _, line := range lines {
		if _, err := filter.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	return console.String()
}

// una sesion como la del log real: vinculacion, conexion, y despues el estado
// y las sesiones UDP repitiendose cada pocos segundos
func TestConsoleFilterSesionCompleta(t *testing.T) {
	session := []string{lineAutoRun, lineSecret, lineSecret}
	for range 5 {
		session = append(session, lineClaim)
	}
	session = append(session, lineApprove, lineApprove, lineApproved, lineIPv6Fail, linePingFail, linePong, lineNotSetup)
	for range 20 {
		session = append(session, lineNoTunnels, lineUDPAuth, lineUDPReady)
	}
	for range 20 {
		session = append(session, lineOneTunnel, lineUDPAuth, lineUDPReady)
	}

	want := "[!] Playit: abre este link para vincular el agente a tu cuenta: https://playit.gg/claim/f0895b07ae\n" +
		"[+] Playit: agente vinculado a tu cuenta.\n" +
		"[!] Playit: conectado, pero tu cuenta no tiene túneles. Crea uno para Minecraft Java desde el panel de playit.gg.\n" +
		"[+] Playit: conectado, 1 túnel(es) activo(s).\n"

	if got := filterAll(t, session...); got != want {
		t.Errorf("la consola recibió:\n%s\nse esperaba:\n%s", got, want)
	}
}

func TestConsoleFilterMuestraCadaLinkNuevo(t *testing.T) {
	otherClaim := strings.Replace(lineClaim, "f0895b07ae", "1ea87a7541", 1)

	got := filterAll(t, lineClaim, lineClaim, otherClaim)
	if strings.Count(got, "playit.gg/claim/") != 2 {
		t.Errorf("cada link distinto tiene que mostrarse una vez:\n%s", got)
	}
}

func TestConsoleFilterEntiendeLineasConColores(t *testing.T) {
	colored := "\x1b[2m2026-09-28T01:58:52.000000Z\x1b[0m \x1b[32m INFO\x1b[0m \x1b[2mplayit_cli::ui\x1b[0m\x1b[2m:\x1b[0m Visit link to setup https://playit.gg/claim/f0895b07ae\n"

	if got := filterAll(t, colored); !strings.Contains(got, "https://playit.gg/claim/f0895b07ae") {
		t.Errorf("no reconoció el link en una línea con colores: %q", got)
	}
}

func TestConsoleFilterIPv4SinConexionSiSeMuestra(t *testing.T) {
	ipv4Fail := strings.Replace(lineIPv6Fail, "addr=[2001:db8::1]:5525", "addr=198.51.100.1:5525", 1)

	if got := filterAll(t, lineIPv6Fail, linePingFail); got != "" {
		t.Errorf("el fallo esperable de IPv6 y su error genérico no deberían mostrarse: %q", got)
	}

	got := filterAll(t, ipv4Fail, linePingFail)
	if !strings.HasPrefix(got, "[-] Playit: failed to send initial ping") {
		t.Errorf("si falla IPv4 no hay conexión y tiene que verse: %q", got)
	}
	if !strings.Contains(got, "failed to ping tunnel server") {
		t.Errorf("después de un fallo de IPv4 el error genérico tiene que verse: %q", got)
	}
}

// sin el intento por IPv6 justo antes, no hay forma de saber que es inofensivo
func TestConsoleFilterErrorGenericoSueltoSeMuestra(t *testing.T) {
	if got := filterAll(t, linePingFail); got != "[-] Playit: failed to ping tunnel server\n" {
		t.Errorf("got %q", got)
	}
}

func TestConsoleFilterSesionSinArmarEsParteDelArranque(t *testing.T) {
	if got := filterAll(t, lineNotSetup); got != "" {
		t.Errorf("SessionNotSetup es del handshake inicial, no debería mostrarse: %q", got)
	}
}

func TestConsoleFilterAvisosRepetidosUnaSolaVez(t *testing.T) {
	expired1 := "2026-09-28T02:00:00.000000Z  WARN playit_agent_core::agent_control::maintained_control: session expired reason=SessionNotFound id=111\n"
	expired2 := "2026-09-28T02:30:00.000000Z  WARN playit_agent_core::agent_control::maintained_control: session expired reason=SessionNotFound id=222\n"

	got := filterAll(t, expired1, expired2)
	if strings.Count(got, "session expired") != 1 {
		t.Errorf("el mismo aviso con otros números tiene que mostrarse una vez:\n%s", got)
	}
	if !strings.HasPrefix(got, "[!] Playit: ") {
		t.Errorf("un WARN va con [!]: %q", got)
	}
}

func TestConsoleFilterLineaPartidaEntreEscrituras(t *testing.T) {
	var console bytes.Buffer
	filter := newConsoleFilter(&console)

	half := len(lineClaim) / 2
	filter.Write([]byte(lineClaim[:half]))
	if console.Len() != 0 {
		t.Fatal("no debería mostrar nada hasta tener la línea completa")
	}
	filter.Write([]byte(lineClaim[half:]))
	if !strings.Contains(console.String(), "playit.gg/claim/f0895b07ae") {
		t.Errorf("no procesó la línea al completarse: %q", console.String())
	}
}

func TestConsoleFilterMuestraLoQueNoEntiende(t *testing.T) {
	panicLine := "thread 'main' panicked at src/main.rs:10:5\n"

	if got := filterAll(t, panicLine); got != "[*] Playit: "+panicLine {
		t.Errorf("una salida fuera de formato tiene que verse: %q", got)
	}
}

// si informara menos bytes de los recibidos, io.MultiWriter cortaria tambien
// la copia al log
func TestConsoleFilterNoCortaAlMultiWriter(t *testing.T) {
	var console, log bytes.Buffer
	writer := io.MultiWriter(newConsoleFilter(&console), &ansiStripper{w: &log})

	n, err := writer.Write([]byte(lineUDPAuth))
	if err != nil || n != len(lineUDPAuth) {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if console.Len() != 0 {
		t.Errorf("el ruido de UDP no debería llegar a la consola: %q", console.String())
	}
	if log.String() != lineUDPAuth {
		t.Errorf("el log tiene que guardar todo: %q", log.String())
	}
}
