package approot

import (
	"os"
	"path/filepath"
	"strings"
)

var root = resolve()

// Dir devuelve el directorio de datos del programa
func Dir() string {
	return root
}

// Path arma una ruta dentro del directorio de datos
func Path(elements ...string) string {
	return filepath.Join(append([]string{root}, elements...)...)
}

// Resolve ancla una ruta relativa al directorio de datos; las absolutas y las vacias quedan como estan
func Resolve(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return Path(path)
}

func resolve() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	if evaluated, err := filepath.EvalSymlinks(executable); err == nil {
		executable = evaluated
	}
	return rootFor(filepath.Dir(executable))
}

// rootFor decide la raiz a partir del directorio del ejecutable. con `go run` y
// `go test` el binario vive en el build cache (un go-build* temporal): ahi se
// usa el directorio actual, que es donde esta el codigo
func rootFor(executableDir string) string {
	if executableDir == "" || executableDir == "." {
		return "."
	}
	if strings.Contains(filepath.ToSlash(executableDir), "/go-build") {
		return "."
	}
	return executableDir
}
