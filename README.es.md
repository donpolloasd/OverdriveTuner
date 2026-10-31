# OverdriveTuner

[English](README.md) · **Español**

Optimizador de PC para gaming en Windows 11, hecho en Go sin dependencias externas. Escanea el
hardware, aplica ajustes reversibles, muestra un overlay de FPS y ayuda a instalar programas.

> **Estado: beta (2.10.1).** Hecho y probado principalmente en una PC con Ryzen 5 5600G + RX 7600 (juegos de Steam, EA, Epic y Minecraft). Algunas funciones no se probaron en otras configuraciones (por ejemplo NVIDIA o Intel). Todos los ajustes se pueden revertir.

> Creado con **Claude Sonnet 5.5 & Claude Opus 5.5** (Anthropic), con la dirección y las pruebas de su autor (Donpolloasd).

## Qué hace

- **Ajustes de rendimiento** (más de 80) con revertir uno por uno o todos. Los que pueden afectar a un
  anti-cheat piden doble confirmación.
- **Diagnóstico de fluidez**: Hz del monitor (se detecta el máximo), gráfica conectada, memoria, red, discos.
- **Overlay de FPS** (proceso aparte, no se inyecta en el juego): FPS, 1 % low, CPU, GPU, RAM, temperaturas
  y aviso de sobrecalentamiento. Detecta solo el juego abierto (Steam, Epic, EA, GOG, Ubisoft, Xbox…).
- **Modo juego**: lista de programas que se cierran al abrir un juego.
- **Liberar RAM**, limpieza de temporales, administrador de programas de inicio.
- **Instalar programas** con winget (launchers, utilidades, monitoreo, periféricos…).
- **Chequeos de hardware**: Resizable BAR, antigüedad del driver, salud y desgaste de discos, espacio libre, guía de BIOS.
- **Ping a servidores de juegos** (latencia, variación, pérdida) y **mantenimiento programado** (limpieza y punto de restauración semanales).
- **Benchmark** antes/después, historial, exportar/importar la configuración.
- Interfaz en español e inglés.

## Cómo funciona el overlay (y por qué no toca el juego)

Los FPS se leen de [RTSS](https://www.guru3d.com/page/rtss/) (solo lectura de su memoria compartida) y
las temperaturas del procesador de [Core Temp](https://www.alcpu.com/CoreTemp/). El Setup puede
instalar los dos y el overlay los abre en segundo plano. OverdriveTuner **no instala drivers de kernel
ni inyecta código** en ningún juego; lo que se mete en el juego es RTSS, no OverdriveTuner.

## Descargar

Bajá `OverdriveTuner-Setup.exe` (instalador) o `OverdriveTuner-Portable.exe` desde [Releases](../../releases) y comprobá el SHA-256 con `SHA256.txt`.

## Actualizaciones automáticas

El Setup tiene una opción (activada por defecto) para instalar solas las versiones nuevas. Si está activa, la
app consulta el último Release estable de GitHub (solo `api.github.com` y las descargas de GitHub; no se envía
nada tuyo), baja `OverdriveTuner-Portable.exe`, lo verifica con el SHA-256 de `SHA256.txt` del mismo Release y
reemplaza el programa instalado; la versión nueva se usa al reiniciar. Si la desactivás (en el Setup o en
Configuración), no se consulta nada y la app se queda en su versión hasta que apliques
`OverdriveTuner-Patch.exe`. Los Releases deben tener el tag `vX.Y.Z` e incluir `OverdriveTuner-Portable.exe` y
`SHA256.txt`. Los binarios no tienen firma digital: la confianza es esta cuenta de GitHub más el SHA-256.

## Compilar

Requisitos: Go 1.24 o superior y bash (en Windows, Git Bash o WSL).

```bash
./build.sh
```

Hace `go vet`, corre las pruebas y genera en `dist/` el instalador, el parche y la versión portable para
Windows 64 bits. Para probar la interfaz fuera de Windows (datos simulados):

```bash
go run ./cmd/fpsforge --demo
```

## Estructura

- `cmd/fpsforge`: el programa (servidor local + ventana propia con WebView2, el motor web que trae Windows; Edge en modo aplicación como respaldo).
- `cmd/setup`: instalador y parche (la misma interfaz, con `-X main.patchMode=1` para el parche).
- `internal/`: módulos (ajustes, escaneo, overlay, RTSS, apps, memoria, actualizaciones, etc.).

## Avisos

- Los ejecutables **no están firmados digitalmente**, así que Windows Defender puede marcarlos como
  sospechosos (por ejemplo con una detección genérica por aprendizaje automático). Podés revisar el código
  y compilarlos vos mismo.
- Pide permisos de administrador porque cambia ajustes del sistema.
- Usalo bajo tu responsabilidad; ver la licencia.

## Licencia

MIT. Ver [LICENSE](LICENSE) y [THIRD_PARTY.md](THIRD_PARTY.md).
