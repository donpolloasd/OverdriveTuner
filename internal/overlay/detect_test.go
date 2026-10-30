package overlay

import (
	"testing"
	"time"
)

func TestPickGame(t *testing.T) {
	out := "\"System\",\"4\",\"Services\",\"0\",\"8 K\"\r\n\"FC26.exe\",\"100\",\"Console\",\"1\",\"3 GB\"\r\n\"chrome.exe\",\"5\",\"Console\",\"1\",\"1 GB\"\r\n"
	run := parseTasklist(out)
	if g := pickGame(run, nil, "", "", "", ""); g != "FC26.exe" {
		t.Fatalf("debía reconocer FC26.exe, fue %q", g)
	}
	if g := pickGame(run, []string{"chrome.exe"}, "", "", "", ""); g != "chrome.exe" {
		t.Fatalf("lo que el usuario agregó tiene prioridad, fue %q", g)
	}
	only := parseTasklist("\"chrome.exe\",\"5\",\"Console\",\"1\",\"1 GB\"\r\n")
	if g := pickGame(only, nil, "", "", "", "msedge.exe"); g != "" {
		t.Fatalf("un navegador a pantalla completa no es un juego, fue %q", g)
	}
	if g := pickGame(only, nil, "", "", "", "miJuego.exe"); g != "miJuego.exe" {
		t.Fatalf("un programa desconocido a pantalla completa sí cuenta, fue %q", g)
	}
	if g := pickGame(only, nil, "", "", "", "mi juego.exe"); g != "" {
		t.Fatalf("nombres no válidos se ignoran, fue %q", g)
	}
}

func TestAutoDetectStartsAndStops(t *testing.T) {
	e, _, _, _, _ := newEng(t)
	cur := ""
	e.d.FindGame = func() string { return cur }
	e.Tick()
	e.Tick()
	if e.gameRun {
		t.Fatal("sin juego no debe medir")
	}
	cur = "FC26.exe"
	e.Tick()
	e.Tick()
	if !e.gameRun || e.pmGame != "FC26.exe" {
		t.Fatalf("debía medir FC26.exe, gameRun=%v game=%q", e.gameRun, e.pmGame)
	}
	cur = ""
	e.Tick()
	e.Tick()
	if e.gameRun {
		t.Fatal("al cerrar el juego debe dejar de medir")
	}
}

func TestPickGameFromRTSS(t *testing.T) {
	none := parseTasklist("")
	if g := pickGame(none, nil, "desconocido.exe", "", "", ""); g != "desconocido.exe" {
		t.Fatalf("el programa que mide RTSS es el juego, fue %q", g)
	}
	if g := pickGame(none, nil, "chrome.exe", "", "", ""); g != "" {
		t.Fatalf("un navegador no es un juego, fue %q", g)
	}
}

func TestLauncherGamePath(t *testing.T) {
	yes := []string{
		`D:\SteamLibrary\steamapps\common\Counter-Strike Global Offensive\game\bin\win64\cs2.exe`,
		`C:\Program Files\Epic Games\Fortnite\FortniteGame\Binaries\Win64\FortniteClient-Win64-Shipping.exe`,
		`C:\Program Files (x86)\Steam\steamapps\common\Hades\x64\Hades.exe`,
		`C:\Program Files\EA Games\EA SPORTS FC 26\FC26.exe`,
	}
	no := []string{
		`C:\Program Files\Epic Games\Launcher\Portal\Binaries\Win64\EpicGamesLauncher.exe`,
		`C:\Program Files (x86)\Steam\steamapps\common\wallpaper_engine\wallpaper64.exe`,
		`D:\SteamLibrary\steamapps\common\Hades\UnityCrashHandler64.exe`,
		`D:\SteamLibrary\steamapps\common\Game\EasyAntiCheat\EasyAntiCheat_EOS_Setup.exe`,
		`C:\Users\yo\Downloads\cs2.exe`,
		`C:\Program Files\Epic Games\Fortnite\readme.txt`,
	}
	for _, p := range yes {
		if !isLauncherGamePath(p) {
			t.Errorf("debía reconocerse: %s", p)
		}
	}
	for _, p := range no {
		if isLauncherGamePath(p) {
			t.Errorf("no debía reconocerse: %s", p)
		}
	}
}

func TestExtraGameDirs(t *testing.T) {
	if !isLauncherGamePath(`E:\Juegos\Fortnite\FortniteGame\Binaries\Win64\FortniteClient-Win64-Shipping.exe`, `E:\Juegos\Fortnite`) {
		t.Fatal("una carpeta de Epic en otro disco debía reconocerse")
	}
	if isLauncherGamePath(`E:\Otra\app.exe`, `E:\Juegos`) {
		t.Fatal("fuera de las carpetas dadas no es un juego")
	}
	if isLauncherGamePath(`C:\Windows\notepad.exe`, `C:\`) {
		t.Fatal("una raíz de disco no debe contar como carpeta de juegos")
	}
	if got := parseEpicManifest([]byte(`{"FormatVersion":0,"InstallLocation":"E:\\Juegos\\Fortnite","AppName":"x"}`)); got != `E:\Juegos\Fortnite` {
		t.Fatalf("manifiesto de Epic mal leído: %q", got)
	}
}

func TestCleanDirsYRegistro(t *testing.T) {
	got := cleanDirs([]string{`"D:\Games\FIFA 26\"`, `d:\games\fifa 26`, `C:\`, ``})
	if len(got) != 1 || got[0] != `D:\Games\FIFA 26` {
		t.Fatalf("cleanDirs = %q", got)
	}
	if !isLauncherGamePath(`D:\Games\FIFA 26\FC26.exe`, got...) {
		t.Fatal("un juego en una carpeta del registro debe reconocerse")
	}
	if isLauncherGamePath(`D:\Games\FIFA 26\Launcher.exe`, got...) {
		t.Fatal("el launcher no es un juego")
	}
}

func TestAlertaYCierre(t *testing.T) {
	if heat(false, 89, 90) || !heat(false, 90, 90) || !heat(true, 88, 90) || heat(true, 86, 90) || heat(false, -1, 90) {
		t.Fatal("la histéresis del aviso está mal")
	}
	got := mergeClose([]string{"chrome.exe", "Steam.exe", "lsass.exe"}, []string{"steam.exe", "cs2.exe"}, "cs2.exe")
	if len(got) != 2 || got[0] != "chrome.exe" || got[1] != "Steam.exe" {
		t.Fatalf("mergeClose = %v", got)
	}
}

func TestGpuBusy(t *testing.T) {
	none := map[string]string{}
	if g := pickGame(none, nil, "", "", "javaw.exe", ""); g != "javaw.exe" {
		t.Errorf("javaw con la placa de video ocupada debería ser el juego, dio %q", g)
	}
	if g := pickGame(none, nil, "", "", "chrome.exe", ""); g != "" {
		t.Errorf("un navegador no es un juego, dio %q", g)
	}
	if gpuBusyScript("javaw.exe") == "" || gpuBusyScript("x'; calc; '.exe") != "" || gpuBusyScript("a b.exe") != "" {
		t.Error("el comando solo debe armarse con nombres seguros")
	}
}

func TestRtssFeedStopTwice(t *testing.T) {
	stop := rtssFeed("x.exe", func(float64) {})
	stop()
	stop() // antes provocaba «close of closed channel»
}

func TestValidExeRechazaCaracteresDePowerShell(t *testing.T) {
	for _, bad := range []string{"a'b.exe", "a$b.exe", "a`b.exe", "a b.exe", `a\b.exe`, "a;b.exe", "x.dll"} {
		if ValidExe(bad) {
			t.Errorf("%q no debería ser válido", bad)
		}
	}
	for _, ok := range []string{"cs2.exe", "javaw.exe", "FC26.exe", "my-game_v2.exe"} {
		if !ValidExe(ok) {
			t.Errorf("%q debería ser válido", ok)
		}
	}
}

func TestExclusiveFullscreenNotice(t *testing.T) {
	e, clk, _, _, dir := newEng(t)
	cur, excl := "", true
	e.d.FindGame = func() string { return cur }
	e.d.Exclusive = func() bool { return excl }
	e.Tick()
	if st, _ := LoadState(dir); st.Exclusive {
		t.Fatal("sin juego abierto no se avisa de pantalla completa exclusiva")
	}
	cur = "cs2.exe"
	e.Tick() // se detecta en el primer segundo
	st, _ := LoadState(dir)
	if !e.gameRun || !st.Exclusive || st.ExclusiveSince == 0 {
		t.Fatalf("debía avisar de pantalla completa exclusiva: gameRun=%v estado=%+v", e.gameRun, st)
	}
	if !ClaimExclusiveNotice(dir, st) || ClaimExclusiveNotice(dir, st) {
		t.Fatal("el aviso de la bandeja sale una sola vez por cada vez que pasa")
	}
	excl = false
	e.Tick()
	if st, _ := LoadState(dir); st.Exclusive {
		t.Fatal("en ventana sin bordes no hay aviso")
	}
	clk.t = clk.t.Add(time.Minute)
	excl = true
	e.Tick()
	st, _ = LoadState(dir)
	if !ClaimExclusiveNotice(dir, st) {
		t.Fatal("si vuelve a pasar, se avisa de nuevo")
	}
}
