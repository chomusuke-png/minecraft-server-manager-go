package playit

import "io"

// ansiStripper saca las secuencias de color que playit escribe aunque su
// salida no sea una terminal. En la consola se ven como colores, pero en
// playit.log quedan como basura alrededor del link de vinculacion
type ansiStripper struct {
	w     io.Writer
	state int
}

const (
	stateText = iota
	stateEscape
	stateCSI
)

// Write filtra con una maquina de estados porque una secuencia puede llegar
// partida entre dos llamadas. Devuelve siempre len(p): si informara menos,
// io.MultiWriter lo tomaria como escritura corta y cortaria tambien la copia
// a la consola
func (s *ansiStripper) Write(p []byte) (int, error) {
	clean := make([]byte, 0, len(p))
	for _, b := range p {
		switch s.state {
		case stateText:
			if b == 0x1b {
				s.state = stateEscape
				continue
			}
			clean = append(clean, b)
		case stateEscape:
			// ESC [ abre una secuencia CSI; cualquier otra cosa despues de ESC
			// es una secuencia de dos bytes que tambien se descarta
			if b == '[' {
				s.state = stateCSI
			} else {
				s.state = stateText
			}
		case stateCSI:
			// los parametros y separadores van entre 0x20 y 0x3f, y la
			// secuencia termina con un byte entre 0x40 y 0x7e
			if b >= 0x40 && b <= 0x7e {
				s.state = stateText
			}
		}
	}

	if _, err := s.w.Write(clean); err != nil {
		return 0, err
	}
	return len(p), nil
}
