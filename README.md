<div align="center">

# OverdriveTuner

**A gaming PC optimizer for Windows 10/11: reversible tweaks, an FPS overlay that never touches your game, and clear diagnostics.**

[![Latest release](https://img.shields.io/github/v/release/donpolloasd/OverdriveTuner?label=release&color=111111)](https://github.com/donpolloasd/OverdriveTuner/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-111111)](LICENSE)
![Windows 10/11](https://img.shields.io/badge/Windows-10%20%7C%2011-111111)
![Go](https://img.shields.io/badge/Go-standard%20library%20only-111111)

**English** · [Español](README.es.md)

[**Download**](https://github.com/donpolloasd/OverdriveTuner/releases/latest) · [Features](#features) · [Safety](#safety-first) · [Install](#install) · [Build](#build-from-source) · [Contact](#contact)

</div>

> [!NOTE]
> **Beta (2.10.1).** Built and tested mainly on one PC (Ryzen 5 5600G + RX 7600; Steam, EA, Epic and Minecraft games). Some features have not been tested on other setups yet (for example NVIDIA or Intel). Every tweak can be reverted. The interface is in Spanish with an English translation (switch it in Settings).

> Created with **Claude Sonnet 5.5 & Claude Opus 5.5** (Anthropic), directed and tested by its author (Donpolloasd).

---

## Features

### Optimize
- **80+ performance tweaks** for FPS, latency, graphics, power, memory, network, services, privacy and Windows 11 debloat. Each one shows its risk level and has a **«?»** button that explains what it changes.
- **One-click profiles** (Recommended and Competitive) that only include safe tweaks for your hardware.
- **Search** any tweak or section with **Ctrl+K**.

### In-game overlay
- FPS, 1% low, frame-time graph, CPU, GPU, VRAM, RAM, temperatures and an overheating alert.
- **Detects the running game on its own** (Steam, Epic, EA, GOG, Ubisoft, Xbox, Riot…).
- **Fully customizable:** any color, ready-made themes, transparent background, corners, border, font, position, monitor, row order, frame time, clock and saved styles.

### Diagnose
- Monitor refresh rate, which GPU the monitor is plugged into, RAM speed (XMP/EXPO) and channels, power plan, Resizable BAR, GPU driver age, disk health and a BIOS guide for your hardware.
- **Ping to game servers**, DNS test and network usage per program.
- **Benchmark** before/after with a stutter detector.

### Tools
- **Game mode:** closes the programs you choose when a game starts.
- **Install programs** with winget (launchers, runtimes, monitoring tools…).
- **Free RAM**, temp-file cleanup, startup manager and scheduled maintenance.

## Safety first

| | |
|---|---|
| **Everything is reversible** | Each tweak saves the previous value and turns back with the same switch. «Restore all» undoes everything, and a restore point is created before applying a profile. |
| **No kernel drivers, no injection** | The overlay is a separate window. FPS are read from [RTSS](https://www.guru3d.com/page/rtss/) and CPU temperature from [Core Temp](https://www.alcpu.com/CoreTemp/), read-only. |
| **Anti-cheat aware** | Tweaks that lower protections required by kernel anti-cheats (HVCI, Secure Boot…) ask for a **double confirmation** and are never part of a profile. |
| **No telemetry** | No accounts, no ads. The only automatic connection is the optional update check to GitHub. See [PRIVACIDAD.md](PRIVACIDAD.md). |

## Install

1. Download from the [**latest release**](https://github.com/donpolloasd/OverdriveTuner/releases/latest):
   - `OverdriveTuner-Setup.exe`: new installation.
   - `OverdriveTuner-Patch.exe`: updates an existing installation and keeps your tweaks and backups.
   - `OverdriveTuner-Portable.exe`: no installation.
2. Optionally check the file against `SHA256.txt`.
3. Run it and accept the administrator prompt (it changes system settings).

> [!IMPORTANT]
> The executables are **not code-signed yet**, so Windows SmartScreen may show *«Windows protected your PC»*: click **More info → Run anyway**. Defender may also raise a generic false positive. Every release is built from this source code by GitHub Actions, and you can also build it yourself.

### Automatic updates
Optional (on by default in the installer, switchable in Settings). The app checks the latest stable GitHub Release, downloads `OverdriveTuner-Portable.exe`, verifies its SHA-256 against the Release's `SHA256.txt` and installs it; the new version is used after a restart. When it is off, nothing is checked.

## Build from source

Requirements: **Go 1.24+** and bash (Git Bash or WSL on Windows). No external dependencies.

```bash
./build.sh                         # vet + tests + Setup, Patch and Portable in dist/
go run ./cmd/fpsforge --demo       # try the interface with simulated data
```

<details>
<summary><b>Project layout</b></summary>

- `cmd/fpsforge`: the program. Local server + its own window with WebView2 (the web engine built into Windows; Edge in app mode as a fallback).
- `cmd/setup`: installer and patch (same code; the patch is built with `-X main.patchMode=1`).
- `internal/`: tweaks, hardware scan, overlay, RTSS and Core Temp readers, diagnostics, updater…
- `.github/workflows/release.yml`: builds, optionally signs, and publishes a Release when a `vX.Y.Z` tag is pushed.

</details>

## Contact

Want to collaborate, or to build your own version, a plugin or a mod? Write to **[overdrivetuner.dev@protonmail.com](mailto:overdrivetuner.dev@protonmail.com)**. For bugs and ideas you can also open an [issue](https://github.com/donpolloasd/OverdriveTuner/issues).

## License

[MIT](LICENSE). Third-party notices in [THIRD_PARTY.md](THIRD_PARTY.md). Use at your own risk.
