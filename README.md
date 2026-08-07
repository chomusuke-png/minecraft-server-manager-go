# Minecraft Server Manager

Herramienta de línea de comandos (Windows y Linux) que administra servidores de Minecraft: crea instancias, descarga automáticamente el loader que elijas (Paper, Fabric, Forge o Vanilla), resuelve el Java correcto para cada versión, hace backups del mundo, reinicia el servidor si se cae, y puede exponerlo a internet con un túnel de [Playit.gg](https://playit.gg) sin que tengas que abrir puertos en tu router.

No hace falta tener Go instalado para usarla — descargá el ejecutable ya compilado para Windows o Linux.

## Descarga

Andá a la sección [Releases](../../releases) de este repositorio y descargá el binario de tu plataforma (`.exe` en Windows, sin extensión en Linux) de la última versión. Poné ese archivo solo, en una carpeta vacía dedicada (el programa va a crear ahí mismo `config.json`, `instances/`, `runtimes/` y `backups/` — conviene que no comparta carpeta con otra cosa).

## Requisitos

- Windows o Linux (amd64/arm64). El resto de la administración del servidor (arrancar, backups, mods, EULA) es igual en los dos; lo único que cambia por SO es cómo se maneja el proceso de Playit por debajo.
- Conexión a internet la primera vez que crees o actualices una instancia (para descargar el jar del servidor y, si hace falta, un runtime de Java).
- Java **no es obligatorio tenerlo instalado de antemano**: si no hay uno compatible, el programa te ofrece descargar automáticamente el JDK correcto para esa versión de Minecraft (Windows y Linux, amd64/arm64).
- [Playit](https://playit.gg/download) es opcional, solo si querés exponer el server a internet sin abrir puertos en tu router. El programa también te ofrece descargarlo solo (`playit.exe` en Windows, binario `playit` en Linux). En Linux no abre una ventana propia como en Windows: su salida se ve en la misma consola y además queda guardada en `playit.log`.

## Primer uso

1. Ejecutá el `.exe`. La primera vez se crea un `config.json` con valores por defecto (más abajo se explica qué es cada campo).
2. Vas a ver el selector de instancias. Como todavía no hay ninguna, elegí `C` para crear una.
3. Te va a pedir:
   - **Nombre** de la instancia (sin espacios, se usa como nombre de carpeta).
   - **RAM** asignada en GB (Enter para usar el valor por defecto de `config.json`).
4. Como todavía no tiene el jar del servidor, te pregunta si querés descargarlo automáticamente. Si decís que sí:
   - Versión de Minecraft (ej. `1.20.1`).
   - Tipo de servidor: **Paper**, **Fabric**, **Forge** o **Vanilla**.
   - Si el Java que tenés no es compatible con esa versión, te ofrece conseguir uno (descarga automática de [Adoptium](https://adoptium.net/) o indicar la ruta a un Java que ya tengas instalado).
5. Configurás `server.properties` la primera vez: MOTD, dificultad, jugadores máximos, `online-mode` y **puerto**. Todo con Enter para aceptar el valor por defecto que se muestra entre corchetes.
6. Si tenés Playit (o aceptaste descargarlo), se lanza y queda conectado a tu cuenta. En Windows en su propia ventana; en Linux, en la misma consola (y logueado en `playit.log`).
7. Aceptás el EULA de Mojang (obligatorio para que el servidor arranque).
8. El servidor arranca. Podés escribir comandos de consola de Minecraft directamente en esa misma terminal (`stop`, `say hola`, etc.) — se reenvían al proceso del servidor. `Ctrl+C` hace un apagado prolijo (guarda el mundo antes de cerrar).

## El menú principal

```
==============================
   SELECTOR DE INSTANCIAS
==============================
1) mi_server    [paper 1.20.1 | 4GB RAM | puerto 25565]
C) crear nueva instancia
U) actualizar loader de una instancia
Q) salir
```

- Un número selecciona esa instancia y la arranca.
- `C` crea una instancia nueva.
- `U` te deja elegir una instancia existente y cambiarle la versión de Minecraft, el tipo de loader, la RAM, el puerto y el mínimo de backups a conservar, sin tener que borrarla y crearla de nuevo.
- `Q` sale del programa.

## `config.json`

Vive al lado del ejecutable y se genera solo la primera vez:

| Campo | Qué hace |
|---|---|
| `java_path` | Java que se usa por defecto si una instancia no tiene uno propio. `"java"` a secas usa el que esté en el `PATH`. |
| `jar_name` | Nombre del jar del servidor dentro de cada instancia (por defecto `server.jar`). |
| `ram_gb` | RAM por defecto para instancias nuevas, en GB. |
| `playit_path` | Dónde está (o se va a descargar) el binario de Playit (`playit.exe` en Windows, `playit` en Linux). |
| `backup_retention_days` | Cuántos días se conservan los backups del mundo antes de borrarse automáticamente. |
| `backup_keep_min` | Piso mínimo de backups que se conservan siempre, sin importar cuántos días de retención hayan pasado (por defecto 3). |

Cada instancia puede pisar su propia RAM, puerto, versión de Java y mínimo de backups sin tocar este archivo global (se guarda en su `instance.json`).

## Qué hace automáticamente

- **Backups**: antes de cada arranque, si la instancia ya tiene mundo, comprime todas las carpetas de dimensión que existan (`world`, `world_nether`, `world_the_end`) en un único zip dentro de `backups/<instancia>/`. Los backups más viejos que `backup_retention_days` se borran solos, pero nunca se baja del piso mínimo `backup_keep_min` (por instancia o global), sin importar la antigüedad.
- **Mods client-only**: en instancias Fabric, escanea la carpeta `mods/` y deshabilita (`.jar` → `.jar.disabled`) los mods marcados como exclusivos de cliente, para que no rompan el arranque del servidor.
- **Reinicio automático**: si el servidor se cae de forma abrupta (no por vos), se reinicia solo a los 10 segundos (cancelable con `Ctrl+C`). Si detecta que el problema fue una versión de Java incompatible, te ofrece resolverlo ahí mismo antes de reintentar.
- **Túnel de Playit**: si tenés el binario configurado, se comparte un único agente entre todas las instancias que tengas corriendo al mismo tiempo — no se abre uno por cada servidor, y se cierra solo cuando cerrás la última instancia que lo estaba usando.

## Compilar desde el código fuente

Necesitás [Go 1.25+](https://go.dev/dl/):

```bash
# Windows
go build -o minecraft-manager.exe ./cmd

# Linux
go build -o minecraft-manager ./cmd

# Cross-compilar Windows desde Linux (o al revés) sin instalar nada más:
GOOS=windows GOARCH=amd64 go build -o minecraft-manager.exe ./cmd
GOOS=linux   GOARCH=amd64 go build -o minecraft-manager     ./cmd
```

No tiene dependencias externas — solo librería estándar de Go.

## To-do

Cosas planeadas, todavía sin implementar:

- [ ] Actualizador automático de la herramienta.
- [ ] Desactivador de mods de cliente para Forge (hoy el escaneo de mods client-only solo cubre instancias Fabric).

## Licencia

MIT — ver [LICENSE](./LICENSE).
