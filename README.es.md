<div align="center">

# OverdriveTuner

**Un optimizador de PC para gaming en Windows 10/11: ajustes reversibles, un overlay de FPS que nunca toca tu juego y diagnósticos claros.**

[![Última versión](https://img.shields.io/github/v/release/donpolloasd/OverdriveTuner?label=versi%C3%B3n&color=111111)](https://github.com/donpolloasd/OverdriveTuner/releases/latest)
[![Licencia: MIT](https://img.shields.io/badge/licencia-MIT-111111)](LICENSE)
![Windows 10/11](https://img.shields.io/badge/Windows-10%20%7C%2011-111111)
![Go](https://img.shields.io/badge/Go-solo%20biblioteca%20est%C3%A1ndar-111111)

[English](README.md) · **Español**

[**Descargar**](https://github.com/donpolloasd/OverdriveTuner/releases/latest) · [Funciones](#funciones) · [Seguridad](#la-seguridad-primero) · [Instalación](#instalación) · [Compilar](#compilar-desde-el-código)

</div>

> [!NOTE]
> **Beta (2.10.1).** Hecho y probado principalmente en una PC (Ryzen 5 5600G + RX 7600; juegos de Steam, EA, Epic y Minecraft). Algunas funciones todavía no se probaron en otras configuraciones (por ejemplo NVIDIA o Intel). Todos los ajustes se pueden revertir. La interfaz está en español con traducción al inglés (se cambia en Configuración).

> Creado con **Claude Sonnet 5.5 & Claude Opus 5.5** (Anthropic), con la dirección y las pruebas de su autor (Donpolloasd).

---

## Funciones

### Optimizar
- **Más de 80 ajustes de rendimiento** para FPS, latencia, gráficos, energía, memoria, red, servicios, privacidad y debloat de Windows 11. Cada uno muestra su nivel de riesgo y tiene un botón **«?»** que explica qué cambia.
- **Perfiles de un clic** (Recomendado y Competitivo) que solo incluyen ajustes seguros para tu hardware.
- **Buscá** cualquier ajuste o sección con **Ctrl+K**.

### Overlay en juego
- FPS, 1% low, gráfico de tiempos de cuadro, CPU, GPU, VRAM, RAM, temperaturas y aviso de sobrecalentamiento.
- **Detecta solo el juego que abriste** (Steam, Epic, EA, GOG, Ubisoft, Xbox, Riot…).
- **Totalmente personalizable:** cualquier color, temas listos, fondo transparente, esquinas, borde, letra, posición, monitor, orden de los datos, tiempo de cuadro, hora y estilos guardados.

### Diagnosticar
- Frecuencia del monitor, a qué gráfica está conectado, velocidad (XMP/EXPO) y canales de la RAM, plan de energía, Resizable BAR, antigüedad del driver de video, salud de los discos y una guía de BIOS para tu hardware.
- **Ping a servidores de juegos**, prueba de DNS y uso de red por programa.
- **Benchmark** antes/después con detector de microcortes.

### Herramientas
- **Modo juego:** cierra los programas que elijas cuando abrís un juego.
- **Instalar programas** con winget (launchers, runtimes, herramientas de monitoreo…).
- **Liberar RAM**, limpieza de temporales, administrador de inicio y mantenimiento programado.

## La seguridad primero

| | |
|---|---|
| **Todo es reversible** | Cada ajuste guarda el valor anterior y vuelve atrás con el mismo interruptor. «Restaurar todo» deshace todo, y antes de aplicar un perfil se crea un punto de restauración. |
| **Sin drivers de kernel ni inyección** | El overlay es una ventana aparte. Los FPS se leen de [RTSS](https://www.guru3d.com/page/rtss/) y la temperatura del procesador de [Core Temp](https://www.alcpu.com/CoreTemp/), solo lectura. |
| **Pensado para los anti-cheat** | Los ajustes que bajan protecciones que exigen los anti-cheat de kernel (HVCI, Secure Boot…) piden **doble confirmación** y nunca forman parte de un perfil. |
| **Sin telemetría** | Sin cuentas ni publicidad. La única conexión automática es la búsqueda opcional de actualizaciones en GitHub. Ver [PRIVACIDAD.md](PRIVACIDAD.md). |

## Instalación

1. Descargá desde la [**última versión**](https://github.com/donpolloasd/OverdriveTuner/releases/latest):
   - `OverdriveTuner-Setup.exe`: instalación nueva.
   - `OverdriveTuner-Patch.exe`: actualiza una instalación existente y conserva tus ajustes y copias de seguridad.
   - `OverdriveTuner-Portable.exe`: sin instalar.
2. Si querés, comprobá el archivo con `SHA256.txt`.
3. Ejecutalo y aceptá el aviso de administrador (cambia configuraciones del sistema).

> [!IMPORTANT]
> Los ejecutables **todavía no tienen firma digital**, así que Windows SmartScreen puede mostrar *«Windows protegió su PC»*: tocá **Más información → Ejecutar de todas formas**. Defender también puede dar un falso positivo genérico. Cada versión se compila desde este código con GitHub Actions, y también podés compilarla vos.

### Actualizaciones automáticas
Opcionales (activadas por defecto en el instalador, se cambian en Configuración). El programa consulta la última versión estable en GitHub, descarga `OverdriveTuner-Portable.exe`, comprueba su SHA-256 con el `SHA256.txt` de esa versión y la instala; la versión nueva se usa al reiniciar. Si están desactivadas, no se consulta nada.

## Compilar desde el código

Requisitos: **Go 1.24 o más nuevo** y bash (Git Bash o WSL en Windows). Sin dependencias externas.

```bash
./build.sh                         # vet + pruebas + Setup, Patch y Portable en dist/
go run ./cmd/fpsforge --demo       # probar la interfaz con datos simulados
```

<details>
<summary><b>Estructura del proyecto</b></summary>

- `cmd/fpsforge`: el programa. Servidor local + ventana propia con WebView2 (el motor web que trae Windows; Edge en modo aplicación como respaldo).
- `cmd/setup`: instalador y parche (el mismo código; el parche se compila con `-X main.patchMode=1`).
- `internal/`: ajustes, escaneo del hardware, overlay, lectores de RTSS y Core Temp, diagnóstico, actualizador…
- `.github/workflows/release.yml`: compila, firma (opcional) y publica una versión cuando se sube un tag `vX.Y.Z`.

</details>

## Licencia

[MIT](LICENSE). Avisos de terceros en [THIRD_PARTY.md](THIRD_PARTY.md). Usalo bajo tu responsabilidad.
