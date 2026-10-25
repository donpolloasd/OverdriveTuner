package tweaks

// Traducciones al inglés de los ajustes de la versión 2.1 (hechas por Claude, sin revisión nativa).
func init() {
	for id, t := range map[string]EnText{
		"net.ipv4pref":   {"Prefer IPv4 over IPv6", "Windows tries IPv4 first. On some networks it shortens connection time and avoids slow IPv6 routes.", "Results vary by internet provider: try it and compare your ping.", ""},
		"net.noipv6":     {"Turn off IPv6", "Turns IPv6 off completely. Useful if your network handles it badly.", "Some services and IPv6-only networks would stop working. It excludes 'Prefer IPv4'.", ""},
		"net.noteredo":   {"Turn off Teredo", "Teredo is an IPv6-over-IPv4 tunnel that almost nobody uses and that sometimes adds latency when connecting to games.", "Some Xbox games and Microsoft Store apps use Teredo for peer-to-peer multiplayer.", ""},
		"pr.wpbt":        {"Turn off WPBT (firmware programs at boot)", "Stops the motherboard firmware from installing and running programs inside Windows at every boot (the mechanism some vendors use for their utilities).", "", ""},
		"pr.devmeta":     {"Do not download device companion apps", "Windows stops looking online for the manufacturer's info and apps when you plug in a peripheral.", "", ""},
		"db.razer":       {"Block automatic Razer software installation", "When you connect a Razer peripheral, Windows does not install Razer Synapse or its service on its own.", "If you use Razer Synapse, install it manually from their site: this only blocks the automatic installation.", ""},
		"db.logi":        {"Block Logitech Download Assistant", "Stops and blocks the Logitech download assistant that runs at every sign-in.", "It does not affect G HUB or drivers that are already installed.", ""},
		"ui.endtask":     {"End task with right click on the taskbar", "Adds 'End task' to the taskbar app menu: close a frozen game without opening Task Manager.", "", ""},
		"ui.lockscreen":  {"Turn off the lock screen", "When you turn on or unlock the PC you go straight to sign-in, without the photo and clock screen.", "", ""},
		"ui.notifs":      {"Turn off notifications and the calendar center", "No Windows pop-ups or notification center while you play.", "You will not see app alerts (mail, messages) until you turn it back on.", ""},
		"ux.fileext":     {"Show file extensions", "You see the .exe, .zip or .jpg of every file. It helps you avoid opening a disguised file.", "", ""},
		"ux.hidden":      {"Show hidden files and folders", "File Explorer stops hiding folders such as AppData.", "", ""},
		"ux.dark":        {"Dark theme for Windows and apps", "Switches the Windows and supported apps color mode to dark.", "", ""},
		"ux.numlock":     {"Num Lock on at startup", "The numeric keypad stays on when you enter Windows.", "", ""},
		"ux.scrollbars":  {"Scroll bars always visible", "Scroll bars do not hide when you are not using them.", "", ""},
		"ux.longpaths":   {"Allow long paths (over 260 characters)", "Compatible programs can use long file paths without errors.", "", ""},
		"ux.battery":     {"Battery percentage in the taskbar", "Shows the percentage next to the battery icon.", "", ""},
		"ux.bsod":        {"Blue screen with technical details", "If Windows crashes, the blue screen shows the error code instead of the sad face.", "", ""},
		"ux.verbose":     {"Detailed messages at sign-in and shutdown", "When you sign in and shut down, Windows shows what it is doing at each step.", "", ""},
		"ux.homegallery": {"Remove 'Home' and 'Gallery' from File Explorer", "File Explorer opens on 'This PC' and without the Home and Gallery entries.", "", ""},
		"upd.pause": {"Defer Windows updates until the year 3000",
			"A pause in Windows Update's routine: it postpones automatic updates until the year 3000 and stops the orchestrator that schedules them. Your system keeps exactly the configuration you chose, with no surprises or unexpected restarts, until you decide to resume them with a single click. Windows Defender definitions are not touched.",
			"Without updates you get no security patches. Windows may re-enable the service after a big update: the program checks this when it opens and warns you. Kernel anti-cheats may require an up-to-date Windows.",
			"Windows updates (including security patches)"},
	} {
		enCatalog[id] = t
	}
	enCategories["comfort"] = EnCategory{"Look and comfort", "Explorer, keyboard and theme details (off by default)"}
	enCategories["updates"] = EnCategory{"Updates", "You decide when Windows updates"}
}
