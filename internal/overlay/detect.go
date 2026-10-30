package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fpsforge/internal/fpslog"
	"fpsforge/internal/rtss"
	"fpsforge/internal/system"
)

// knownGames son juegos que se reconocen por el nombre del programa, aunque estén en ventana.
var knownGames = []string{
	"cs2.exe", "csgo.exe", "fc26.exe", "fc25.exe", "fc24.exe", "eurotrucks2.exe", "amtrucks.exe",
	"valorant-win64-shipping.exe", "fortniteclient-win64-shipping.exe", "r5apex.exe", "rocketleague.exe",
	"gta5.exe", "gta5_enhanced.exe", "dota2.exe", "overwatch.exe", "destiny2.exe", "rainbowsix.exe",
	"rainbowsix_vulkan.exe", "tslgame.exe", "escapefromtarkov.exe", "cyberpunk2077.exe", "eldenring.exe",
	"hl2.exe", "left4dead2.exe", "witcher3.exe", "rdr2.exe", "bf2042.exe", "bfv.exe", "cod.exe",
	"modernwarfare.exe", "marvel-win64-shipping.exe", "fh5.exe", "forzahorizon5.exe", "minecraft.windows.exe",
}

// notGames son programas que pueden ocupar toda la pantalla sin ser un juego.
var notGames = map[string]bool{
	"explorer.exe": true, "applicationframehost.exe": true, "searchhost.exe": true, "shellexperiencehost.exe": true,
	"startmenuexperiencehost.exe": true, "lockapp.exe": true, "dwm.exe": true, "chrome.exe": true, "msedge.exe": true,
	"firefox.exe": true, "brave.exe": true, "opera.exe": true, "vlc.exe": true, "spotify.exe": true, "obs64.exe": true,
	"discord.exe": true, "teams.exe": true, "ms-teams.exe": true, "code.exe": true, "mstsc.exe": true, "steamwebhelper.exe": true,
	"steam.exe": true, "epicgameslauncher.exe": true, "overdrivetuner.exe": true, "overdrivetuner-portable.exe": true,
	"powerpnt.exe": true, "winword.exe": true, "excel.exe": true, "vmware.exe": true, "vmware-vmx.exe": true, "zoom.exe": true,
}

// IsNotGame dice si el programa es de uso general (navegador, escritorio, etc.).
func IsNotGame(exe string) bool { return notGames[strings.ToLower(exe)] }

// parseTasklist devuelve los nombres de programa (en minúsculas → nombre original) de «tasklist /FO CSV /NH».
func parseTasklist(out string) map[string]string {
	m := map[string]string{}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, `"`) {
			continue
		}
		end := strings.Index(ln[1:], `"`)
		if end < 0 {
			continue
		}
		name := ln[1 : 1+end]
		m[strings.ToLower(name)] = name
	}
	return m
}

// pickGame elige el juego entre los programas en ejecución: primero los que el usuario configuró,
// después los conocidos y, si no hay ninguno, el programa que ocupa toda la pantalla (si lo hay).
func pickGame(running map[string]string, user []string, rtssGame, pathGame, gpuGame, fullscreen string) string {
	for _, g := range user {
		if n, ok := running[strings.ToLower(g)]; ok && ValidExe(n) {
			return n
		}
	}
	for _, g := range knownGames {
		if n, ok := running[g]; ok && ValidExe(n) {
			return n
		}
	}
	if rtssGame != "" && ValidExe(rtssGame) && !IsNotGame(rtssGame) {
		return rtssGame
	}
	if pathGame != "" && ValidExe(pathGame) && !IsNotGame(pathGame) {
		return pathGame
	}
	if fullscreen != "" && ValidExe(fullscreen) && !IsNotGame(fullscreen) {
		return fullscreen
	}
	if gpuGame != "" && ValidExe(gpuGame) && !IsNotGame(gpuGame) {
		return gpuGame
	}
	return ""
}

var safeExeName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// gpuBusyScript arma el comando de PowerShell que suma el uso del motor 3D de la placa de video del
// programa. Devuelve "" si el nombre tiene caracteres raros (no se arma el comando).
func gpuBusyScript(exe string) string {
	name := strings.TrimSuffix(exe, filepath.Ext(exe))
	if !safeExeName.MatchString(name) {
		return ""
	}
	return "$ids=@(Get-Process -Name '" + name + "' -ErrorAction SilentlyContinue | ForEach-Object Id); if(-not $ids){0;exit}; " +
		"$t=0; (Get-Counter '\\GPU Engine(*engtype_3D)\\Utilization Percentage' -ErrorAction SilentlyContinue).CounterSamples | " +
		"ForEach-Object { if($_.InstanceName -match 'pid_(\\d+)_' -and $ids -contains [int]$matches[1]){ $t+=$_.CookedValue } }; [int]$t"
}

// gpuBusyMin es el uso mínimo (%) del motor 3D para considerar que el programa activo es un juego.
const gpuBusyMin = 15

var (
	gpuMu    sync.Mutex
	gpuExe   string
	gpuStamp time.Time
	gpuRes   string
	gpuBusy  bool // hay una consulta de PowerShell en curso
)

// gpuBusyGame devuelve el programa de la ventana activa si está usando la placa de video (3D) con fuerza:
// así se reconocen los juegos en ventana (como Minecraft con Java) sin conocer su nombre. Se consulta
// como mucho una vez cada 8 segundos por programa. La consulta (PowerShell, tarda unos segundos) corre
// aparte: mientras tanto se devuelve el último resultado y el overlay sigue funcionando sin trabarse.
func gpuBusyGame(sys system.System, running map[string]string) string {
	gpuMu.Lock()
	if gpuRes != "" { // ya se reconoció: se mantiene mientras el programa siga abierto (aunque cambies de ventana)
		if _, ok := running[strings.ToLower(gpuRes)]; ok {
			r := gpuRes
			gpuMu.Unlock()
			return r
		}
		gpuRes, gpuExe = "", ""
	}
	gpuMu.Unlock()
	exe := foregroundAnyExe()
	if exe == "" || !ValidExe(exe) || IsNotGame(exe) {
		return ""
	}
	gpuMu.Lock()
	defer gpuMu.Unlock()
	if gpuBusy || (strings.EqualFold(exe, gpuExe) && time.Since(gpuStamp) < 8*time.Second) {
		return gpuRes
	}
	gpuExe, gpuStamp, gpuRes = exe, time.Now(), ""
	script := gpuBusyScript(exe)
	if script == "" {
		return ""
	}
	gpuBusy = true
	go func() {
		out, _, err := sys.Exec(12*time.Second, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
		gpuMu.Lock()
		defer gpuMu.Unlock()
		gpuBusy = false
		if err != nil || !strings.EqualFold(exe, gpuExe) {
			return
		}
		if v, e := strconv.Atoi(strings.TrimSpace(out)); e == nil && v >= gpuBusyMin {
			gpuRes = exe
			fpslog.Diag("detección automática: %s usa la placa de video (%d %%), se toma como juego", exe, v)
		}
	}()
	return ""
}

// rtssActiveGame devuelve el programa que RTSS (MSI Afterburner) está midiendo ahora, si hay uno.
// Es la misma señal que usa Afterburner para saber qué juego es: no hace falta conocer su nombre.
func rtssActiveGame() string {
	defer func() { _ = recover() }()
	now := rtss.Tick()
	for _, e := range rtss.Read() {
		if _, _, ok := e.FPS(now); !ok {
			continue
		}
		n := strings.ReplaceAll(e.Name, "/", `\`)
		if i := strings.LastIndex(n, `\`); i >= 0 {
			n = n[i+1:]
		}
		if ValidExe(n) && !IsNotGame(n) {
			return n
		}
	}
	return ""
}

var listFailed bool

// FindGame busca el juego que está abierto («» si no hay). Es lo que usa la detección automática.
func FindGame(sys system.System, userGames func() ([]string, []string)) string {
	running, ok := runningExes()
	if !ok { // respaldo: tasklist.exe (más lento)
		out, _, err := sys.Exec(8*time.Second, "tasklist.exe", "/NH", "/FO", "CSV")
		if err != nil {
			if !listFailed {
				listFailed = true
				fpslog.Diag("detección automática: no se pudo leer la lista de programas: %v", err)
			}
			return ""
		}
		running = parseTasklist(out)
	}
	var user, extraDirs []string
	if userGames != nil {
		user, extraDirs = userGames()
	}
	rg, lg, fs := rtssActiveGame(), launcherGame(append(append(epicDirs(), registryDirs()...), extraDirs...)), foregroundFullscreenExe()
	g := pickGame(running, user, rg, lg, "", fs)
	if g == "" {
		g = pickGame(running, user, rg, lg, gpuBusyGame(sys, running), fs)
	}
	return g
}

// gameDirs son carpetas donde los launchers instalan juegos (en minúsculas, con barras invertidas).
var gameDirs = []string{
	`\steamapps\common\`, `\epic games\`, `\ea games\`, `\gog galaxy\games\`, `\gog games\`,
	`\ubisoft game launcher\games\`, `\xboxgames\`, `\riot games\`, `\battle.net\games\`,
}

// notGameDirs son carpetas de esos mismos lugares que no son juegos (los launchers y sus herramientas).
var notGameDirs = []string{
	`\epic games\launcher\`, `\epic games\epicgameslauncher\`, `\epic games\directxredist\`,
	`\steamapps\common\steamworks shared\`, `\steamapps\common\steamvr\`, `\steamapps\common\wallpaper_engine\`,
	`\steamapps\common\steam linux runtime`, `\steamapps\common\proton`, `\_commonredist\`, `\riot client\`,
}

// notGameWords son partes del nombre que delatan programas auxiliares (informes de fallos, anti-cheat, instaladores).
var notGameWords = []string{
	"crashhandler", "crashreport", "crashpad", "cefsubprocess", "easyanticheat", "beservice", "battleye", "unins", "uninst",
	"setup", "redist", "vcredist", "dxsetup", "installer", "launcher", "updater", "webhelper", "overlay", "bootstrap",
}

// isLauncherGamePath dice si la ruta de un programa está en la carpeta de juegos de un launcher
// (Steam, Epic, EA, GOG, Ubisoft, Xbox, Riot) y no es un programa auxiliar. Así se reconocen todos los
// juegos instalados desde ahí sin tener que conocer sus nombres.
func isLauncherGamePath(path string, extra ...string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "/", `\`))
	in := false
	for _, d := range gameDirs {
		if strings.Contains(p, d) {
			in = true
			break
		}
	}
	for _, d := range extra { // carpetas de juegos propias (Epic en otro disco, carpetas que agregó el usuario)
		d = strings.ToLower(strings.TrimRight(strings.ReplaceAll(strings.TrimSpace(d), "/", `\`), `\`))
		if len(d) >= 4 && strings.HasPrefix(p, d+`\`) {
			in = true
			break
		}
	}
	if !in {
		return false
	}
	for _, d := range notGameDirs {
		if strings.Contains(p, d) {
			return false
		}
	}
	name := p[strings.LastIndex(p, `\`)+1:]
	if !strings.HasSuffix(name, ".exe") {
		return false
	}
	for _, w := range notGameWords {
		if strings.Contains(name, w) {
			return false
		}
	}
	return true
}

// parseEpicManifest devuelve la carpeta de instalación de un archivo .item del launcher de Epic.
func parseEpicManifest(b []byte) string {
	var m struct{ InstallLocation string }
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	return strings.TrimSpace(m.InstallLocation)
}

var (
	epicMu    sync.Mutex
	epicList  []string
	epicStamp time.Time
)

var (
	regMu    sync.Mutex
	regList  []string
	regStamp time.Time
)

// registryDirs lista las carpetas de juegos que EA, GOG y Ubisoft guardan en el registro (se relee cada minuto).
func registryDirs() []string {
	regMu.Lock()
	defer regMu.Unlock()
	if time.Since(regStamp) < time.Minute {
		return regList
	}
	regStamp = time.Now()
	regList = cleanDirs(registryGameDirs())
	return regList
}

// cleanDirs normaliza las rutas leídas del registro: sin comillas, sin barra final y sin repetidos.
func cleanDirs(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range in {
		d = strings.TrimRight(strings.Trim(strings.TrimSpace(d), `"`), `\/`)
		k := strings.ToLower(d)
		if len(d) >= 4 && !seen[k] {
			seen[k] = true
			out = append(out, d)
		}
	}
	return out
}

// epicDirs lista las carpetas donde Epic instaló juegos, en cualquier disco (lee los manifiestos que
// el launcher guarda en ProgramData). Se vuelve a leer cada minuto.
func epicDirs() []string {
	epicMu.Lock()
	defer epicMu.Unlock()
	if time.Since(epicStamp) < time.Minute {
		return epicList
	}
	epicStamp = time.Now()
	epicList = nil
	pd := os.Getenv("ProgramData")
	if pd == "" {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests", "*.item"))
	for _, f := range files {
		if b, err := os.ReadFile(f); err == nil {
			if d := parseEpicManifest(b); d != "" {
				epicList = append(epicList, d)
			}
		}
	}
	return epicList
}
