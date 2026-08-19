# Minecraft Server Manager

Herramienta de línea de comandos (Windows y Linux) que administra servidores de Minecraft: crea instancias, descarga automáticamente el loader que elijas (Paper, Fabric, Forge, NeoForge o Vanilla), resuelve el Java correcto para cada versión, hace backups del mundo, reinicia el servidor si se cae, y puede exponerlo a internet con un túnel ([Playit.gg](https://playit.gg) o [ngrok](https://ngrok.com), a elección por instancia) sin que tengas que abrir puertos en tu router.

No hace falta tener Go instalado para usarla — descargá el ejecutable ya compilado para Windows o Linux.

## Descarga

Andá a la sección [Releases](../../releases) de este repositorio y descargá el binario de tu plataforma (`.exe` en Windows, sin extensión en Linux) de la última versión. Poné ese archivo solo, en una carpeta vacía dedicada (el programa va a crear ahí mismo `config.json`, `instances/`, `runtimes/` y `backups/` — conviene que no comparta carpeta con otra cosa).

## Requisitos

- Windows o Linux (amd64/arm64). El resto de la administración del servidor (arrancar, backups, mods, EULA) es igual en los dos; lo único que cambia por SO es cómo se maneja el proceso del túnel por debajo.
- Conexión a internet la primera vez que crees o actualices una instancia (para descargar el jar del servidor y, si hace falta, un runtime de Java).
- Java **no es obligatorio tenerlo instalado de antemano**: si no hay uno compatible, el programa te ofrece descargar automáticamente el JDK correcto para esa versión de Minecraft (Windows y Linux, amd64/arm64).
- El túnel es opcional y se elige por instancia (Playit, ngrok o ninguno), solo si querés exponer el server a internet sin abrir puertos en tu router:
  - **[Playit](https://playit.gg/download)** (recomendado): el programa te ofrece descargarlo solo (`playit.exe` en Windows, binario `playit` en Linux). No necesita cuenta pre-configurada — al arrancar por primera vez te muestra un link para vincularlo a tu cuenta. En Linux no abre una ventana propia como en Windows: su salida se ve en la misma consola y además queda guardada en `playit.log`.
  - **[ngrok](https://ngrok.com/download)**: a diferencia de Playit, siempre necesita una cuenta y un token — sacalo de [dashboard.ngrok.com/get-started/your-authtoken](https://dashboard.ngrok.com/get-started/your-authtoken) y ponelo en `ngrok_authtoken` dentro de `config.json` antes de usarlo. Cada instancia que lo use lanza su propio proceso (a diferencia de Playit, que comparte uno solo); la URL pública queda en `ngrok.log` dentro de la instancia y también se imprime en consola al arrancar.

## Primer uso

1. Ejecutá el `.exe`. La primera vez se crea un `config.json` con valores por defecto (más abajo se explica qué es cada campo).
2. Vas a ver el selector de instancias. Como todavía no hay ninguna, elegí `C` para crear una.
3. Te va a pedir:
   - **Nombre** de la instancia (sin espacios, se usa como nombre de carpeta).
   - **RAM** asignada en GB (Enter para usar el valor por defecto de `config.json`).
   - **Túnel**: Playit (recomendado, Enter), ngrok o ninguno.
4. Como todavía no tiene el jar del servidor, te pregunta si querés descargarlo automáticamente. Si decís que sí:
   - Versión de Minecraft (ej. `1.20.1`).
   - Tipo de servidor: **Paper**, **Fabric**, **Forge**, **NeoForge** o **Vanilla**.
   - Si el Java que tenés no es compatible con esa versión, te ofrece conseguir uno (descarga automática de [Adoptium](https://adoptium.net/) o indicar la ruta a un Java que ya tengas instalado).
5. Configurás `server.properties` la primera vez: MOTD, dificultad, tipo de mundo (normal, plano, biomas grandes o amplificado), jugadores máximos, `online-mode` y **puerto**. Todo con Enter para aceptar el valor por defecto que se muestra entre corchetes.
6. Si la instancia usa un túnel (o aceptaste descargarlo), se lanza. Con Playit, en Windows en su propia ventana y en Linux en la misma consola (logueado en `playit.log`). Con ngrok, la URL pública se imprime en consola y queda en `ngrok.log` dentro de la instancia.
7. Aceptás el EULA de Mojang (obligatorio para que el servidor arranque).
8. El servidor arranca. Podés escribir comandos de consola de Minecraft directamente en esa misma terminal (`stop`, `say hola`, etc.) — se reenvían al proceso del servidor. `Ctrl+C` hace un apagado prolijo (guarda el mundo antes de cerrar).

## El menú principal

```
============================================================
  MINECRAFT SERVER MANAGER                            v1.2.0
  por chomusuke-png (Zumito)
============================================================

  INSTANCIAS
    1) mi_server    [paper 1.20.1 | 4GB RAM | puerto 25565]

  ACCIONES
    C) crear nueva instancia
    U) actualizar loader de una instancia
    D) borrar una instancia
    Q) salir
```

- Un número selecciona esa instancia y la arranca.
- `C` crea una instancia nueva.
- `U` te deja elegir una instancia existente y cambiarle la versión de Minecraft, el tipo de loader, la RAM, el puerto, el túnel y el mínimo de backups a conservar, sin tener que borrarla y crearla de nuevo.
- `D` te deja elegir una instancia y borrarla por completo (mundo, backups, todo). Pide escribir el nombre exacto para confirmar; cualquier otra cosa (o Enter en blanco) cancela sin tocar nada.
- `Q` sale del programa.

## `config.json`

Vive al lado del ejecutable y se genera solo la primera vez:

| Campo | Qué hace |
|---|---|
| `java_path` | Java que se usa por defecto si una instancia no tiene uno propio. `"java"` a secas usa el que esté en el `PATH`. |
| `jar_name` | Nombre del jar del servidor dentro de cada instancia (por defecto `server.jar`). |
| `ram_gb` | RAM por defecto para instancias nuevas, en GB. |
| `playit_path` | Dónde está (o se va a descargar) el binario de Playit (`playit.exe` en Windows, `playit` en Linux). |
| `ngrok_path` | Dónde está (o se va a descargar) el binario de ngrok (`ngrok.exe` en Windows, `ngrok` en Linux). |
| `ngrok_authtoken` | Token de cuenta de ngrok, sacado de [dashboard.ngrok.com/get-started/your-authtoken](https://dashboard.ngrok.com/get-started/your-authtoken). Vacío por defecto; sin esto ninguna instancia puede usar ngrok como túnel. |
| `backup_retention_days` | Cuántos días se conservan los backups del mundo antes de borrarse automáticamente. |
| `backup_keep_min` | Piso mínimo de backups que se conservan siempre, sin importar cuántos días de retención hayan pasado (por defecto 3). |
| `disable_update_check` | `true` para no chequear versión nueva al arrancar. Por defecto (`false`/ausente) chequea siempre. |

Cada instancia puede pisar su propia RAM, puerto, versión de Java, mínimo de backups y túnel sin tocar este archivo global (se guarda en su `instance.json`).

## Qué hace automáticamente

- **Backups**: antes de cada arranque, si la instancia ya tiene mundo, comprime todas las carpetas de dimensión que existan (`world`, `world_nether`, `world_the_end`) en un único zip dentro de `backups/<instancia>/`. Los backups más viejos que `backup_retention_days` se borran solos, pero nunca se baja del piso mínimo `backup_keep_min` (por instancia o global), sin importar la antigüedad.
- **Mods client-only**: escanea la carpeta `mods/` y deshabilita (`.jar` → `.jar.disabled`) los mods marcados como exclusivos de cliente, para que no rompan el arranque del servidor. En Fabric usa el campo `environment` de `fabric.mod.json`. En Forge y NeoForge no existe un campo oficial equivalente: se usa la misma convención que herramientas como ServerPackCreator, un mod que se autodeclara como su propia dependencia con `side="CLIENT"` en `mods.toml`/`neoforge.mods.toml`. Sin esa autodeclaración no hay forma confiable de saberlo, así que esos mods se dejan sin tocar. Si algún mod se detecta mal, se puede proteger agregándolo a `mods_whitelist.txt` (se genera solo en la raíz de la instancia): un nombre de archivo `.jar` por línea, con o sin extensión, sin importar mayúsculas. Para el caso inverso —un mod de cliente que no se autodeclara y por eso no se detecta— está `mods_blacklist.txt`, con el mismo formato: lo que listes ahí se deshabilita siempre, sin mirar el `.jar`. Si un mod aparece en las dos listas, gana la whitelist.
- **Reinicio automático**: si el servidor se cae de forma abrupta (no por vos), se reinicia solo a los 10 segundos (cancelable con `Ctrl+C`). Si detecta que el problema fue una versión de Java incompatible, te ofrece resolverlo ahí mismo antes de reintentar.
- **Túnel**: se elige por instancia (`tunnel_provider` en `instance.json`, editable desde el menú de actualización). Sin ese campo (instancias creadas antes de que existiera esta opción) se sigue tratando como Playit, para no cambiar el comportamiento que ya tenían. Con Playit, si tenés el binario configurado, se comparte un único agente entre todas las instancias que lo usen al mismo tiempo — no se abre uno por cada servidor, y se cierra solo cuando cerrás la última instancia que lo estaba usando. Con ngrok cada instancia lanza su propio proceso, porque el puerto se le pasa por línea de comandos en cada arranque en vez de configurarse del lado de la cuenta como en Playit.
- **Actualización de la herramienta**: al arrancar, chequea contra los [Releases](../../releases) de este repositorio si hay una versión más nueva. Si la hay, pregunta antes de hacer nada; si aceptás, descarga el binario correspondiente (verificando su checksum contra el digest que publica GitHub) y lo deja instalado en el lugar del actual — el ejecutable viejo queda al lado como `.old` por si el nuevo no arranca. Hace falta reiniciar la herramienta para que tome el cambio. Se puede desactivar con `disable_update_check`. Una build compilada a mano sin el flag de versión (ver más abajo) nunca chequea, porque no tiene con qué comparar.

## Compilar desde el código fuente

Necesitás [Go 1.25+](https://go.dev/dl/):

```bash
# Windows
go build -o builds/msm-windows-amd64.exe ./cmd

# Linux
go build -o builds/msm-linux-amd64 ./cmd

# Cross-compilar Windows desde Linux (o al revés) sin instalar nada más:
GOOS=windows GOARCH=amd64 go build -o builds/msm-windows-amd64.exe ./cmd
GOOS=linux   GOARCH=amd64 go build -o builds/msm-linux-amd64       ./cmd
```

Una build así queda identificada como `dev` y nunca va a ofrecer actualizarse sola (no tiene versión contra la cual comparar). Los releases oficiales se compilan pisando esa versión:

```bash
go build -ldflags "-X main.version=v1.2.0" -o builds/msm-windows-amd64.exe ./cmd
```

## Licencia

MIT — ver [LICENSE](./LICENSE).
