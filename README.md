# Minecraft Server Manager

Herramienta de línea de comandos para Windows que administra servidores de Minecraft: crea instancias, descarga automáticamente el loader que elijas (Paper, Fabric, Forge o Vanilla), resuelve el Java correcto para cada versión, hace backups del mundo, reinicia el servidor si se cae, y puede exponerlo a internet con un túnel de [Playit.gg](https://playit.gg) sin que tengas que abrir puertos en tu router.

No hace falta tener Go instalado para usarla — descargá el ejecutable ya compilado.

## Descarga

Andá a la sección [Releases](../../releases) de este repositorio y descargá el `.exe` de la última versión. Poné ese archivo solo, en una carpeta vacía dedicada (el programa va a crear ahí mismo `config.json`, `instances/`, `runtimes/` y `backups/` — conviene que no comparta carpeta con otra cosa).

## Requisitos

- Windows (usa `powershell`/`taskkill`/`tasklist` internamente, así que por ahora es Windows-only).
- Conexión a internet la primera vez que crees o actualices una instancia (para descargar el jar del servidor y, si hace falta, un runtime de Java).
- Java **no es obligatorio tenerlo instalado de antemano**: si no hay uno compatible, el programa te ofrece descargar automáticamente el JDK correcto para esa versión de Minecraft.
- [playit.exe](https://playit.gg/download) es opcional, solo si querés exponer el server a internet sin abrir puertos en tu router. El programa también te ofrece descargarlo solo.

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
6. Si tenés `playit.exe` (o aceptaste descargarlo), se lanza en su propia ventana y queda conectado a tu cuenta.
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
- `U` te deja elegir una instancia existente y cambiarle la versión de Minecraft, el tipo de loader, la RAM y el puerto sin tener que borrarla y crearla de nuevo.
- `Q` sale del programa.

## `config.json`

Vive al lado del ejecutable y se genera solo la primera vez:

| Campo | Qué hace |
|---|---|
| `java_path` | Java que se usa por defecto si una instancia no tiene uno propio. `"java"` a secas usa el que esté en el `PATH`. |
| `jar_name` | Nombre del jar del servidor dentro de cada instancia (por defecto `server.jar`). |
| `ram_gb` | RAM por defecto para instancias nuevas, en GB. |
| `playit_path` | Dónde está (o se va a descargar) `playit.exe`. |
| `backup_retention_days` | Cuántos días se conservan los backups del mundo antes de borrarse automáticamente. |

Cada instancia puede pisar su propia RAM, puerto y versión de Java sin tocar este archivo global (se guarda en su `instance.json`).

## Qué hace automáticamente

- **Backups**: antes de cada arranque, si la instancia ya tiene un mundo (`world/`), lo comprime a `backups/world_backup_<fecha>.zip`. Los backups más viejos que `backup_retention_days` se borran solos.
- **Mods client-only**: en instancias Fabric, escanea la carpeta `mods/` y deshabilita (`.jar` → `.jar.disabled`) los mods marcados como exclusivos de cliente, para que no rompan el arranque del servidor.
- **Reinicio automático**: si el servidor se cae de forma abrupta (no por vos), se reinicia solo a los 10 segundos (cancelable con `Ctrl+C`). Si detecta que el problema fue una versión de Java incompatible, te ofrece resolverlo ahí mismo antes de reintentar.
- **Túnel de Playit.gg**: si tenés `playit.exe` configurado, se comparte un único agente entre todas las instancias que tengas corriendo al mismo tiempo — no se abre uno por cada servidor, y se cierra solo cuando cerrás la última instancia que lo estaba usando.

## Compilar desde el código fuente

Si preferís compilarlo vos en vez de bajar el `.exe` de Releases, necesitás [Go 1.25+](https://go.dev/dl/):

```bash
go build -o minecraft-manager.exe ./cmd
```

No tiene dependencias externas — solo librería estándar de Go.
