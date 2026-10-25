package apps

// Traducciones al inglés (hechas por Claude, sin revisión nativa).
var enItems = map[string][2]string{ // id → {descripción, nota}
	"faceit":        {"FACEIT client for competitive matches (CS2).", ""},
	"faceitac":      {"FACEIT's anti-cheat, required to play its matches.", ""},
	"spotify":       {"Music and podcasts.", ""},
	"wiztree":       {"Shows which files take the most space on your disks, in seconds.", ""},
	"malwarebytes":  {"Anti-malware with on-demand scans (real-time protection is paid).", ""},
	"vscode":        {"Microsoft's code editor.", ""},
	"intelDSA":      {"Detects and updates processor, Intel graphics and network drivers (only useful with Intel hardware).", ""},
	"playnite":      {"One library for the games from all your launchers.", ""},
	"zoom":          {"Video calls and meetings.", ""},
	"bitwarden":     {"Open-source password manager.", ""},
	"bcuninstaller": {"Uninstalls programs in bulk and cleans up leftovers.", ""},
	"rufus":         {"Creates bootable USB drives to install Windows.", ""},
	"ventoy":        {"One bootable USB drive with several ISO images.", ""},
	"sumatra":       {"Very lightweight PDF reader.", ""},
	"acrobat":       {"Adobe's PDF reader, for forms and signatures.", ""},
	"libreoffice":   {"Free office suite compatible with Word and Excel.", ""},
	"sharex":        {"Screenshots and screen recording with hotkeys.", ""},
	"streamlabs":    {"Live streaming with alerts and widgets.", ""},
	"audacity":      {"Free audio editor.", ""},
	"handbrake":     {"Converts and compresses videos.", ""},
	"ghub":          {"Software for Logitech G mice, keyboards and headsets.", ""},
	"synapse":       {"Software for Razer peripherals.", ""},
	"steelseries":   {"Software for SteelSeries peripherals.", ""},
	"java":          {"What Minecraft and other Java programs ask for.", ""},
	"crystalinfo":   {"Health and temperature of your disks.", ""},
	"crystalmark":   {"Measures your disks' speed.", ""},
	"occt":          {"Stress and stability tests for processor, memory and graphics card.", ""},
	"latencymon":    {"Finds which driver causes audio dropouts and latency.", ""},
	"cinebench":     {"Processor performance test.", ""},
	"rtss":          {"RivaTuner Statistics Server, without Afterburner. With it open, OverdriveTuner's overlay can read FPS in games where Windows blocks the other method.", ""},
	"git":           {"Version control.", ""},
	"python":        {"Python language for scripts and automation.", ""},
	"node":          {"JavaScript runtime outside the browser.", ""},
	"brave":         {"Fast browser with built-in ad blocking.", ""},
	"firefox":       {"Open-source, privacy-focused browser.", ""},
	"chrome":        {"Google's browser.", ""},
	"steam":         {"Valve's store and library (CS2, ETS2 and thousands more).", ""},
	"epic":          {"Epic's store, with free games every week.", ""},
	"ea":            {"Electronic Arts launcher (EA SPORTS FC, Battlefield…).", ""},
	"ubisoft":       {"Ubisoft launcher.", ""},
	"battlenet":     {"Blizzard launcher.", ""},
	"gog":           {"GOG launcher and unified library.", ""},
	"discord":       {"Voice and text chat for team play.", ""},
	"telegram":      {"Fast cross-platform messaging.", ""},
	"winrar":        {"File compressor (paid, with a trial period).", ""},
	"7zip":          {"Free, open-source file compressor.", ""},
	"vlc":           {"Video and audio player that opens almost anything.", ""},
	"obs":           {"Recording and live streaming.", ""},
	"notepad":       {"Text and code editor.", ""},
	"powertoys":     {"Microsoft utilities for power users.", ""},
	"everything":    {"Instant file search.", ""},
	"qbit":          {"Ad-free torrent client.", ""},
	"vcredist":      {"2005 to 2015-2022, 32 and 64 bit. What a game most often lacks when it errors on launch.", ""},
	"dotnet":        {"Needed by many programs and launchers.", ""},
	"directx":       {"Older DirectX libraries that older games ask for.", ""},
	"hwinfo":        {"Sensors and full hardware data.", ""},
	"cpuz":          {"Processor, motherboard and memory data.", ""},
	"gpuz":          {"Graphics card data and sensors.", ""},
	"afterburner":   {"FPS overlay and graphics card tuning.", ""},
	"gdrive":        {"Syncs your files with Google's cloud.", ""},
	"onedrive":      {"Microsoft's cloud, built into Windows.", ""},
	"anydesk":       {"Remote control of another PC, useful to help or repair computers remotely.", ""},
	"teamspeak":     {"Low-latency voice chat for team play.", ""},
	"whatsapp":      {"WhatsApp messaging on the PC.", ""},
	"medal":         {"Records and shares clips of your games.", ""},
	"lossless":      {"Upscales the image and generates extra frames in games (paid, bought on Steam).", ""},
	"coretemp":      {"Processor temperature. With it open, the OverdriveTuner overlay can show the CPU temperature.", ""},
	"furmark":       {"Graphics card stress test: shows temperature and stability under load.", ""},
	"ryzenmaster":   {"Tuning and monitoring for Ryzen processors.", ""},
	"amdsoft":       {"Control panel and drivers for Radeon graphics cards.", ""},
	"nvapp":         {"Control panel, drivers and settings for GeForce graphics cards.", ""},
	"intelgfx":      {"Control panel and drivers for Intel graphics (Arc and integrated).", ""},
	"ddu":           {"Completely removes video drivers. Used from Safe Mode.", "Run it by hand, preferably in Safe Mode."},
}

var enNames = map[string]string{
	"vcredist": "Visual C++ (all versions)", "dotnet": ".NET Desktop Runtime (8, 9 and 10)", "directx": "DirectX (end-user runtime)",
}

var enCats = map[string]string{
	"browsers": "Browsers", "launchers": "Game launchers", "chat": "Communication",
	"tools": "Compressors and utilities", "media": "Capture and multimedia", "periph": "Peripherals", "dev": "Development", "runtimes": "Game runtimes", "hardware": "Hardware and monitoring",
}

// Localize devuelve el programa traducido al inglés.
func (it Item) Localize(lang string) Item {
	if lang != "en" {
		return it
	}
	if e, ok := enItems[it.ID]; ok {
		it.Desc, it.Extra = e[0], e[1]
	}
	if n, ok := enNames[it.ID]; ok {
		it.Name = n
	}
	return it
}

// LocalizeCategories devuelve las categorías en el idioma pedido.
func LocalizeCategories(lang string) []Category {
	cs := Categories()
	if lang == "en" {
		for i := range cs {
			cs[i].Name = enCats[cs[i].ID]
		}
	}
	return cs
}
