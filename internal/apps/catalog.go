// Package apps instala y actualiza programas con winget (el administrador de paquetes de Windows 11).
// Todos los identificadores se verificaron contra el repositorio oficial microsoft/winget-pkgs.
package apps

// Item es un programa instalable.
type Item struct {
	ID    string   `json:"id"`
	Cat   string   `json:"cat"`
	Name  string   `json:"name"`
	Desc  string   `json:"desc"`
	IDs   []string `json:"-"`               // identificadores de winget (puede ser más de uno)
	Extra string   `json:"extra,omitempty"` // nota adicional
	URL   string   `json:"url,omitempty"`   // si existe, no se instala con winget: el botón abre esta página oficial
	// Vendor limita el programa al hardware que tiene la PC: gpu:amd, gpu:nvidia, gpu:intel, cpu:amd o
	// intel (procesador o gráfica Intel). Vacío = se muestra siempre.
	Vendor string `json:"vendor,omitempty"`
	// Site es la página oficial de descarga (para cuando winget no puede instalarlo).
	Site string `json:"site,omitempty"`
}

// Category agrupa programas.
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Categories devuelve las categorías en orden.
func Categories() []Category {
	return []Category{
		{"browsers", "Navegadores"}, {"launchers", "Launchers de juegos"}, {"chat", "Comunicación"},
		{"tools", "Compresores y utilidades"}, {"media", "Captura y multimedia"}, {"periph", "Periféricos"},
		{"runtimes", "Runtimes para juegos"}, {"hardware", "Hardware y monitoreo"}, {"dev", "Desarrollo"},
	}
}

// Catalog devuelve todos los programas.
func Catalog() []Item {
	l := catalog()
	for i := range l {
		if l[i].URL != "" {
			l[i].Site = l[i].URL
		} else {
			l[i].Site = sites[l[i].ID]
		}
	}
	return l
}

// sites son las páginas oficiales de descarga: se ofrecen cuando winget no puede instalar un programa.
var sites = map[string]string{
	"brave":         "https://brave.com/download/",
	"firefox":       "https://www.mozilla.org/firefox/new/",
	"chrome":        "https://www.google.com/chrome/",
	"steam":         "https://store.steampowered.com/about/",
	"epic":          "https://store.epicgames.com/download",
	"ea":            "https://www.ea.com/ea-app",
	"ubisoft":       "https://ubisoftconnect.com/",
	"battlenet":     "https://www.blizzard.com/download",
	"gog":           "https://www.gog.com/galaxy",
	"faceit":        "https://www.faceit.com/en/download-faceit",
	"faceitac":      "https://www.faceit.com/en/download-faceit",
	"discord":       "https://discord.com/download",
	"telegram":      "https://desktop.telegram.org/",
	"spotify":       "https://www.spotify.com/download/",
	"winrar":        "https://www.win-rar.com/download.html",
	"7zip":          "https://www.7-zip.org/download.html",
	"vlc":           "https://www.videolan.org/vlc/",
	"obs":           "https://obsproject.com/download",
	"notepad":       "https://notepad-plus-plus.org/downloads/",
	"powertoys":     "https://github.com/microsoft/PowerToys/releases",
	"everything":    "https://www.voidtools.com/downloads/",
	"qbit":          "https://www.qbittorrent.org/download",
	"wiztree":       "https://diskanalyzer.com/download",
	"malwarebytes":  "https://www.malwarebytes.com/mwb-download",
	"vscode":        "https://code.visualstudio.com/download",
	"vcredist":      "https://learn.microsoft.com/cpp/windows/latest-supported-vc-redist",
	"dotnet":        "https://dotnet.microsoft.com/download/dotnet",
	"directx":       "https://www.microsoft.com/download/details.aspx?id=35",
	"hwinfo":        "https://www.hwinfo.com/download/",
	"cpuz":          "https://www.cpuid.com/softwares/cpu-z.html",
	"gpuz":          "https://www.techpowerup.com/gpuz/",
	"intelDSA":      "https://www.intel.com/content/www/us/en/support/detect.html",
	"afterburner":   "https://www.msi.com/Landing/afterburner/graphics-cards",
	"ddu":           "https://www.wagnardsoft.com/display-driver-uninstaller-DDU-",
	"playnite":      "https://playnite.link/",
	"zoom":          "https://zoom.us/download",
	"bitwarden":     "https://bitwarden.com/download/",
	"bcuninstaller": "https://www.bcuninstaller.com/",
	"rufus":         "https://rufus.ie/",
	"ventoy":        "https://www.ventoy.net/en/download.html",
	"sumatra":       "https://www.sumatrapdfreader.org/download-free-pdf-viewer",
	"acrobat":       "https://get.adobe.com/reader/",
	"libreoffice":   "https://www.libreoffice.org/download/",
	"sharex":        "https://getsharex.com/downloads",
	"streamlabs":    "https://streamlabs.com/streamlabs-desktop",
	"audacity":      "https://www.audacityteam.org/download/",
	"handbrake":     "https://handbrake.fr/downloads.php",
	"ghub":          "https://www.logitechg.com/innovation/g-hub.html",
	"synapse":       "https://www.razer.com/synapse-4",
	"steelseries":   "https://steelseries.com/gg",
	"java":          "https://adoptium.net/temurin/releases/",
	"crystalinfo":   "https://crystalmark.info/en/software/crystaldiskinfo/",
	"crystalmark":   "https://crystalmark.info/en/software/crystaldiskmark/",
	"occt":          "https://www.ocbase.com/download",
	"latencymon":    "https://www.resplendence.com/latencymon",
	"cinebench":     "https://www.maxon.net/en/cinebench",
	"rtss":          "https://www.guru3d.com/page/rtss/",
	"git":           "https://git-scm.com/downloads",
	"python":        "https://www.python.org/downloads/",
	"node":          "https://nodejs.org/en/download",
	"gdrive":        "https://www.google.com/drive/download/",
	"onedrive":      "https://www.microsoft.com/microsoft-365/onedrive/download",
	"anydesk":       "https://anydesk.com/en/downloads/windows",
	"teamspeak":     "https://www.teamspeak.com/en/downloads/",
	"coretemp":      "https://www.alcpu.com/CoreTemp/",
	"furmark":       "https://geeks3d.com/furmark/downloads/",
}

func catalog() []Item {
	vc := []string{
		"Microsoft.VCRedist.2015+.x64", "Microsoft.VCRedist.2015+.x86",
		"Microsoft.VCRedist.2013.x64", "Microsoft.VCRedist.2013.x86",
		"Microsoft.VCRedist.2012.x64", "Microsoft.VCRedist.2012.x86",
		"Microsoft.VCRedist.2010.x64", "Microsoft.VCRedist.2010.x86",
		"Microsoft.VCRedist.2008.x64", "Microsoft.VCRedist.2008.x86",
		"Microsoft.VCRedist.2005.x64", "Microsoft.VCRedist.2005.x86",
	}
	return []Item{
		{ID: "brave", Cat: "browsers", Name: "Brave", Desc: "Navegador rápido con bloqueo de anuncios incluido.", IDs: []string{"Brave.Brave"}},
		{ID: "firefox", Cat: "browsers", Name: "Mozilla Firefox", Desc: "Navegador de código abierto centrado en la privacidad.", IDs: []string{"Mozilla.Firefox"}},
		{ID: "chrome", Cat: "browsers", Name: "Google Chrome", Desc: "El navegador de Google.", IDs: []string{"Google.Chrome"}},

		{ID: "steam", Cat: "launchers", Name: "Steam", Desc: "Tienda y biblioteca de Valve (CS2, ETS2 y miles más).", IDs: []string{"Valve.Steam"}},
		{ID: "epic", Cat: "launchers", Name: "Epic Games Launcher", Desc: "Tienda de Epic con juegos gratis cada semana.", IDs: []string{"EpicGames.EpicGamesLauncher"}},
		{ID: "ea", Cat: "launchers", Name: "EA app", Desc: "Launcher de Electronic Arts (EA SPORTS FC, Battlefield…).", IDs: []string{"ElectronicArts.EADesktop"}},
		{ID: "ubisoft", Cat: "launchers", Name: "Ubisoft Connect", Desc: "Launcher de Ubisoft.", IDs: []string{"Ubisoft.Connect"}},
		{ID: "battlenet", Cat: "launchers", Name: "Battle.net", Desc: "Launcher de Blizzard.", IDs: []string{"Blizzard.BattleNet"}},
		{ID: "gog", Cat: "launchers", Name: "GOG Galaxy", Desc: "Launcher de GOG y biblioteca unificada.", IDs: []string{"GOG.Galaxy"}},
		{ID: "faceit", Cat: "launchers", Name: "FACEIT", Desc: "Cliente de FACEIT para partidas competitivas (CS2).", IDs: []string{"FACEITLTD.FACEITClient"}},
		{ID: "faceitac", Cat: "launchers", Name: "FACEIT Anti-Cheat", Desc: "El anti-cheat de FACEIT, necesario para jugar sus partidas.", IDs: []string{"FACEITLTD.FACEITAC"},
			Extra: "Es de kernel: por eso otros programas con drivers pueden chocar con él. Se instala con permisos de administrador y suele pedir reiniciar."},

		{ID: "discord", Cat: "chat", Name: "Discord", Desc: "Chat de voz y texto para jugar en equipo.", IDs: []string{"Discord.Discord"},
			Extra: "El overlay de Discord dentro del juego es lo que puede dar problemas: se puede desactivar en sus ajustes."},
		{ID: "telegram", Cat: "chat", Name: "Telegram", Desc: "Mensajería rápida y multiplataforma.", IDs: []string{"Telegram.TelegramDesktop"}},
		{ID: "spotify", Cat: "chat", Name: "Spotify", Desc: "Música y podcasts.", IDs: []string{"Spotify.Spotify"}},

		{ID: "winrar", Cat: "tools", Name: "WinRAR", Desc: "Compresor de archivos (de pago con período de prueba).", IDs: []string{"RARLab.WinRAR"}},
		{ID: "7zip", Cat: "tools", Name: "7-Zip", Desc: "Compresor de archivos gratuito y de código abierto.", IDs: []string{"7zip.7zip"}},
		{ID: "vlc", Cat: "tools", Name: "VLC", Desc: "Reproductor de video y audio que abre casi todo.", IDs: []string{"VideoLAN.VLC"}},
		{ID: "obs", Cat: "tools", Name: "OBS Studio", Desc: "Grabación y transmisión en vivo.", IDs: []string{"OBSProject.OBSStudio"},
			Extra: "La captura de juego de OBS se engancha al juego; algunas versiones de anti-cheat la bloquean. La captura de pantalla no tiene ese problema."},
		{ID: "notepad", Cat: "tools", Name: "Notepad++", Desc: "Editor de texto y código.", IDs: []string{"Notepad++.Notepad++"}},
		{ID: "powertoys", Cat: "tools", Name: "Microsoft PowerToys", Desc: "Utilidades de Microsoft para usuarios avanzados.", IDs: []string{"Microsoft.PowerToys"}},
		{ID: "everything", Cat: "tools", Name: "Everything", Desc: "Búsqueda instantánea de archivos.", IDs: []string{"voidtools.Everything"}},
		{ID: "qbit", Cat: "tools", Name: "qBittorrent", Desc: "Cliente de torrents sin publicidad.", IDs: []string{"qBittorrent.qBittorrent"}},
		{ID: "wiztree", Cat: "tools", Name: "WizTree", Desc: "Muestra qué archivos ocupan más espacio en tus discos, en segundos.", IDs: []string{"AntibodySoftware.WizTree"}},
		{ID: "malwarebytes", Cat: "tools", Name: "Malwarebytes", Desc: "Antimalware con análisis a demanda (la protección en tiempo real es de pago).", IDs: []string{"Malwarebytes.Malwarebytes"},
			Extra: "Si su protección en tiempo real se activa, a veces marca juegos o anti-cheats: agregá excepciones si pasa."},
		{ID: "vscode", Cat: "tools", Name: "Visual Studio Code", Desc: "Editor de código de Microsoft.", IDs: []string{"Microsoft.VisualStudioCode"}},

		{ID: "vcredist", Cat: "runtimes", Name: "Visual C++ (todas las versiones)", Desc: "2005 a 2015-2022, 32 y 64 bits. Lo que más falta cuando un juego da error al abrir.", IDs: vc},
		{ID: "dotnet", Cat: "runtimes", Name: ".NET Desktop Runtime (8, 9 y 10)", Desc: "Necesario para muchos programas y launchers.",
			IDs: []string{"Microsoft.DotNet.DesktopRuntime.8", "Microsoft.DotNet.DesktopRuntime.9", "Microsoft.DotNet.DesktopRuntime.10"}},
		{ID: "directx", Cat: "runtimes", Name: "DirectX (runtime de usuario)", Desc: "Librerías de DirectX antiguas que piden juegos más viejos.", IDs: []string{"Microsoft.DirectX"}},

		{ID: "hwinfo", Cat: "hardware", Name: "HWiNFO", Desc: "Sensores y datos completos del hardware.", IDs: []string{"REALiX.HWiNFO"}},
		{ID: "cpuz", Cat: "hardware", Name: "CPU-Z", Desc: "Datos del procesador, la placa y la memoria.", IDs: []string{"CPUID.CPU-Z"}},
		{ID: "gpuz", Cat: "hardware", Name: "GPU-Z", Desc: "Datos y sensores de la placa de video.", IDs: []string{"TechPowerUp.GPU-Z"}},
		{ID: "intelDSA", Cat: "hardware", Name: "Intel Driver & Support Assistant", Desc: "Detecta y actualiza los drivers de procesador, gráficos Intel y red (solo útil con hardware Intel).", IDs: []string{"Intel.IntelDriverAndSupportAssistant"}, Vendor: "intel"},
		{ID: "afterburner", Cat: "hardware", Name: "MSI Afterburner + RTSS", Desc: "Overlay de FPS y ajuste de la placa de video.", IDs: []string{"Guru3D.Afterburner", "Guru3D.RTSS"},
			Extra: "RTSS inyecta un componente dentro del juego: es el caso más conocido de conflicto con anti-cheat."},
		{ID: "ddu", Cat: "hardware", Name: "Display Driver Uninstaller (DDU)", Desc: "Desinstala por completo los drivers de video. Se usa desde el modo seguro.", IDs: []string{"Wagnardsoft.DisplayDriverUninstaller"},
			Extra: "Se ejecuta a mano, preferentemente en modo seguro."},

		{ID: "playnite", Cat: "launchers", Name: "Playnite", Desc: "Biblioteca única para los juegos de todos tus launchers.", IDs: []string{"Playnite.Playnite"}},
		{ID: "zoom", Cat: "chat", Name: "Zoom", Desc: "Videollamadas y reuniones.", IDs: []string{"Zoom.Zoom"}},

		{ID: "bitwarden", Cat: "tools", Name: "Bitwarden", Desc: "Administrador de contraseñas de código abierto.", IDs: []string{"Bitwarden.Bitwarden"}},
		{ID: "bcuninstaller", Cat: "tools", Name: "Bulk Crap Uninstaller", Desc: "Desinstala programas en lote y limpia sus restos.", IDs: []string{"Klocman.BulkCrapUninstaller"}},
		{ID: "rufus", Cat: "tools", Name: "Rufus", Desc: "Crea pendrives de arranque para instalar Windows.", IDs: []string{"Rufus.Rufus"}},
		{ID: "ventoy", Cat: "tools", Name: "Ventoy", Desc: "Un pendrive de arranque con varias imágenes ISO.", IDs: []string{"Ventoy.Ventoy"}},
		{ID: "sumatra", Cat: "tools", Name: "SumatraPDF", Desc: "Lector de PDF muy liviano.", IDs: []string{"SumatraPDF.SumatraPDF"}},
		{ID: "acrobat", Cat: "tools", Name: "Adobe Acrobat Reader", Desc: "Lector de PDF de Adobe, para formularios y firmas.", IDs: []string{"Adobe.Acrobat.Reader.64-bit"}},
		{ID: "libreoffice", Cat: "tools", Name: "LibreOffice", Desc: "Suite de oficina gratuita compatible con Word y Excel.", IDs: []string{"TheDocumentFoundation.LibreOffice"}},

		{ID: "sharex", Cat: "media", Name: "ShareX", Desc: "Capturas y grabación de pantalla con atajos.", IDs: []string{"ShareX.ShareX"}},
		{ID: "streamlabs", Cat: "media", Name: "Streamlabs", Desc: "Transmisión en vivo con alertas y widgets.", IDs: []string{"Streamlabs.Streamlabs"}},
		{ID: "audacity", Cat: "media", Name: "Audacity", Desc: "Editor de audio gratuito.", IDs: []string{"Audacity.Audacity"}},
		{ID: "handbrake", Cat: "media", Name: "HandBrake", Desc: "Convierte y comprime videos.", IDs: []string{"HandBrake.HandBrake"}},

		{ID: "ghub", Cat: "periph", Name: "Logitech G HUB", Desc: "Software de mouse, teclado y auriculares Logitech G.", IDs: []string{"Logitech.GHUB"}},
		{ID: "synapse", Cat: "periph", Name: "Razer Synapse", Desc: "Software de periféricos Razer.", IDs: []string{"RazerInc.RazerInstaller.Synapse4"}},
		{ID: "steelseries", Cat: "periph", Name: "SteelSeries GG", Desc: "Software de periféricos SteelSeries.", IDs: []string{"SteelSeries.GG"}},

		{ID: "java", Cat: "runtimes", Name: "Java (Temurin 21)", Desc: "Lo que piden Minecraft y otros programas en Java.", IDs: []string{"EclipseAdoptium.Temurin.21.JRE"}},

		{ID: "crystalinfo", Cat: "hardware", Name: "CrystalDiskInfo", Desc: "Salud y temperatura de tus discos.", IDs: []string{"CrystalDewWorld.CrystalDiskInfo"}},
		{ID: "crystalmark", Cat: "hardware", Name: "CrystalDiskMark", Desc: "Mide la velocidad de tus discos.", IDs: []string{"CrystalDewWorld.CrystalDiskMark"}},
		{ID: "occt", Cat: "hardware", Name: "OCCT", Desc: "Pruebas de estrés y estabilidad de procesador, memoria y placa de video.", IDs: []string{"OCBase.OCCT.Personal"}},
		{ID: "latencymon", Cat: "hardware", Name: "LatencyMon", Desc: "Detecta qué driver causa microcortes de audio y latencia.", IDs: []string{"Resplendence.LatencyMon"}},
		{ID: "cinebench", Cat: "hardware", Name: "Cinebench R23", Desc: "Prueba de rendimiento del procesador.", IDs: []string{"Maxon.CinebenchR23"}},
		{ID: "rtss", Cat: "hardware", Name: "RTSS (solo medición de FPS)", Desc: "RivaTuner Statistics Server, sin Afterburner. Con él abierto el overlay de OverdriveTuner puede leer FPS en juegos donde Windows bloquea el otro método.", IDs: []string{"Guru3D.RTSS"}},

		{ID: "git", Cat: "dev", Name: "Git", Desc: "Control de versiones.", IDs: []string{"Git.Git"}},
		{ID: "python", Cat: "dev", Name: "Python 3.13", Desc: "Lenguaje Python para scripts y automatización.", IDs: []string{"Python.Python.3.13"}},
		{ID: "node", Cat: "dev", Name: "Node.js (LTS)", Desc: "Entorno para JavaScript fuera del navegador.", IDs: []string{"OpenJS.NodeJS.LTS"}},
		{ID: "gdrive", Cat: "tools", Name: "Google Drive", Desc: "Sincroniza tus archivos con la nube de Google.", IDs: []string{"Google.GoogleDrive"}},
		{ID: "onedrive", Cat: "tools", Name: "Microsoft OneDrive", Desc: "Nube de Microsoft, integrada con Windows.", IDs: []string{"Microsoft.OneDrive"}},
		{ID: "anydesk", Cat: "tools", Name: "AnyDesk", Desc: "Control remoto de otra PC, útil para ayudar o arreglar equipos a distancia.", IDs: []string{"AnyDesk.AnyDesk"}},
		{ID: "teamspeak", Cat: "chat", Name: "TeamSpeak", Desc: "Chat de voz de baja latencia para jugar en equipo.", IDs: []string{"TeamSpeakSystems.TeamSpeakClient"}},
		{ID: "whatsapp", Cat: "chat", Name: "WhatsApp", Desc: "Mensajería de WhatsApp en la PC.", URL: "https://www.whatsapp.com/download"},
		{ID: "medal", Cat: "media", Name: "Medal", Desc: "Graba y comparte clips de tus partidas.", URL: "https://medal.tv/download"},
		{ID: "lossless", Cat: "media", Name: "Lossless Scaling", Desc: "Escala la imagen y genera cuadros extra en juegos (de pago, se compra en Steam).", URL: "https://store.steampowered.com/app/993090/Lossless_Scaling/"},
		{ID: "coretemp", Cat: "hardware", Name: "Core Temp", Desc: "Temperatura del procesador. Con él abierto, el overlay de OverdriveTuner puede mostrar la temperatura del CPU.", IDs: []string{"ALCPU.CoreTemp"}},
		{ID: "furmark", Cat: "hardware", Name: "FurMark", Desc: "Prueba de estrés de la placa de video: muestra temperatura y estabilidad bajo carga.", IDs: []string{"Geeks3D.FurMark.2"}},
		{ID: "ryzenmaster", Cat: "hardware", Name: "AMD Ryzen Master", Desc: "Ajuste y monitoreo de procesadores Ryzen.", URL: "https://www.amd.com/en/products/software/ryzen-master.html", Vendor: "cpu:amd"},
		{ID: "amdsoft", Cat: "hardware", Name: "AMD Software: Adrenalin Edition", Desc: "Panel y drivers de las placas de video Radeon.", URL: "https://www.amd.com/en/support/download/drivers.html", Vendor: "gpu:amd"},
		{ID: "nvapp", Cat: "hardware", Name: "NVIDIA App", Desc: "Panel, drivers y ajustes de las placas de video GeForce.", URL: "https://www.nvidia.com/en-us/software/nvidia-app/", Vendor: "gpu:nvidia"},
		{ID: "intelgfx", Cat: "hardware", Name: "Intel Graphics Software", Desc: "Panel y drivers de los gráficos Intel (Arc e integrados).", URL: "https://www.intel.com/content/www/us/en/download-center/home.html", Vendor: "intel"},
	}
}

// ByID busca un programa.
func ByID(id string) (Item, bool) {
	for _, it := range Catalog() {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}

// Matches dice si el programa corresponde al hardware (CPU y GPU en minúsculas: amd, intel, nvidia).
func (it Item) Matches(cpuVendor string, gpuVendors []string) bool {
	if it.Vendor == "" {
		return true
	}
	has := func(v string) bool {
		for _, g := range gpuVendors {
			if g == v {
				return true
			}
		}
		return false
	}
	switch it.Vendor {
	case "gpu:amd":
		return has("amd")
	case "gpu:nvidia":
		return has("nvidia")
	case "gpu:intel":
		return has("intel")
	case "cpu:amd":
		return cpuVendor == "amd"
	case "intel":
		return cpuVendor == "intel" || has("intel")
	}
	return true
}

// DriverPage es la página oficial de drivers de cada fabricante de placas de video.
var DriverPage = map[string]string{
	"nvidia": "https://www.nvidia.com/Download/index.aspx",
	"amd":    "https://www.amd.com/en/support/download/drivers.html",
	"intel":  "https://www.intel.com/content/www/us/en/download-center/home.html",
}
