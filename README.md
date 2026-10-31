# OverdriveTuner

**English** · [Español](README.es.md)

A gaming PC optimizer for Windows 11, written in Go with no external dependencies. It scans your
hardware, applies reversible tweaks, shows an FPS overlay and helps you install programs.

> **Status: beta (2.9.0).** Built and tested mainly on one PC (Ryzen 5 5600G + RX 7600; Steam, EA, Epic and Minecraft games). Some features have not been tested on other setups (for example NVIDIA or Intel). Every tweak can be reverted. The interface is in Spanish with an English translation (switch it in Settings); the English text was written by the author and may have mistakes.

> Created with **Claude Sonnet 5.5 & Claude Opus 5.5** (Anthropic), directed and tested by its author.

## What it does

- **Performance tweaks** (80+), revertible one by one or all at once. Tweaks that may affect an
  anti-cheat ask for a double confirmation.
- **Smoothness diagnostics**: monitor Hz (maximum is detected), connected GPU, memory, network, disks.
- **FPS overlay** (separate process, nothing is injected into the game): FPS, 1 % low, CPU, GPU, RAM,
  temperatures and an overheating warning. It detects the running game on its own (Steam, Epic, EA,
  GOG, Ubisoft, Xbox...).
- **Game mode**: a list of programs that are closed when a game starts.
- **Free RAM**, temp-file cleanup, startup program manager.
- **Install programs** with winget (launchers, utilities, monitoring, peripherals...), with a link to
  the official website when winget cannot install something.
- **Health checks**: Resizable BAR, GPU driver age, disk health/wear, free space, BIOS settings guide.
- **Ping to game servers** (latency, jitter, loss) and **scheduled maintenance** (weekly cleanup, weekly restore point).
- **Benchmark** before/after, history, export/import of the configuration.

## How the overlay works (and why it does not touch the game)

FPS are read from [RTSS](https://www.guru3d.com/page/rtss/) (read-only access to its shared memory) and
CPU temperature from [Core Temp](https://www.alcpu.com/CoreTemp/). The installer can install both and
the overlay opens them in the background. OverdriveTuner **installs no kernel drivers and injects no
code** into any game; what hooks into the game is RTSS, not OverdriveTuner.

## Download

Get `OverdriveTuner-Setup.exe` (installer) or `OverdriveTuner-Portable.exe` from the [Releases](../../releases) page and check the SHA-256 in `SHA256.txt`.

## Automatic updates

The installer has an option (on by default) to install new versions automatically. When on, the app checks
the latest stable GitHub Release (only `api.github.com` / GitHub downloads, nothing about you is sent),
downloads `OverdriveTuner-Portable.exe`, verifies it against the SHA-256 in the Release's `SHA256.txt` and
replaces the installed program; the new version is used after a restart. When off (in the installer or in
Settings), nothing is checked and the app stays on its version until you apply `OverdriveTuner-Patch.exe`.
Releases must be tagged `vX.Y.Z` and include `OverdriveTuner-Portable.exe` and `SHA256.txt`. The binaries are
not code-signed: trust comes from this GitHub account plus the SHA-256.

## Build

Requirements: Go 1.24 or newer and bash (on Windows, Git Bash or WSL).

```bash
./build.sh
```

It runs `go vet`, the tests, and produces the installer, the patch and the portable build for 64-bit
Windows in `dist/`. To try the interface outside Windows (simulated data):

```bash
go run ./cmd/fpsforge --demo
```

## Layout

- `cmd/fpsforge`: the program (local server + its own window with WebView2, the web engine built into Windows; Edge in app mode as a fallback).
- `cmd/setup`: installer and patch (same UI, built with `-X main.patchMode=1` for the patch).
- `internal/`: modules (tweaks, scan, overlay, RTSS, apps, memory, update, etc.).

## Warnings

- The executables are **not code-signed**, so Windows Defender may flag them as suspicious (for example
  with a generic machine-learning detection). You can read the code and build them yourself.
- It asks for administrator rights because it changes system settings.
- Use at your own risk; see the license.

## License

MIT. See [LICENSE](LICENSE) and [THIRD_PARTY.md](THIRD_PARTY.md).
