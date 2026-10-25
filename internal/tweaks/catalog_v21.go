package tweaks

// Versión 2.1: ajustes de red, limpieza, comodidad y pausa de actualizaciones.

const (
	tcpip6      = `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters`
	explorerKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer`
	personalize = `HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	wuUX        = `HKLM\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings`
	pauseStart  = "2025-01-01T00:00:00Z"
	pauseEnd    = "3000-01-01T00:00:00Z"
)

// denyDirOps bloquea que un instalador escriba en una carpeta (para frenar instaladores automáticos).
func denyDirOps(dir, killProc string) Op {
	kill := ""
	if killProc != "" {
		kill = "Get-Process -Name '" + killProc + "' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue\n"
	}
	apply := kill + `$p = ` + dir + `
if (Test-Path -LiteralPath $p) { Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue }
New-Item -ItemType Directory -Force -Path $p | Out-Null
icacls $p /deny '*S-1-1-0:(W)' | Out-Null`
	revert := `$p = ` + dir + `
if (Test-Path -LiteralPath $p) {
  icacls $p /remove:d '*S-1-1-0' | Out-Null
  Remove-Item -LiteralPath $p -Force -ErrorAction SilentlyContinue
}`
	return PS(apply, revert)
}

const updTasksApply = `$dir = Join-Path $env:ProgramData 'FPSForge'
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$saved = @()
foreach ($path in @('\Microsoft\Windows\UpdateOrchestrator\','\Microsoft\Windows\WindowsUpdate\')) {
  foreach ($t in @(Get-ScheduledTask -TaskPath $path -ErrorAction SilentlyContinue)) {
    if ($t.State -ne 'Disabled') {
      try { Disable-ScheduledTask -TaskPath $t.TaskPath -TaskName $t.TaskName -ErrorAction Stop | Out-Null; $saved += ($t.TaskPath + $t.TaskName) } catch { }
    }
  }
}
ConvertTo-Json -InputObject @($saved) | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $dir 'upd_tasks.json')
Write-Output ('Tareas de actualización detenidas: ' + $saved.Count)`

const updTasksRevert = `$f = Join-Path $env:ProgramData 'FPSForge\upd_tasks.json'
if (-not (Test-Path -LiteralPath $f)) { exit 0 }
foreach ($full in @(Get-Content -Raw -Encoding UTF8 -LiteralPath $f | ConvertFrom-Json)) {
  $i = $full.LastIndexOf('\')
  try { Enable-ScheduledTask -TaskPath $full.Substring(0, $i + 1) -TaskName $full.Substring($i + 1) -ErrorAction Stop | Out-Null } catch { }
}
Remove-Item -LiteralPath $f -ErrorAction SilentlyContinue`

func v21Tweaks() []Tweak {
	return []Tweak{
		// ---- Red ----
		{ID: "net.ipv4pref", Test: true, Cat: "network", Name: "Preferir IPv4 sobre IPv6",
			Desc: "Windows intenta primero IPv4. En algunas redes baja el tiempo de conexión y evita rutas IPv6 lentas.",
			Warn: "Efecto variable según tu proveedor de internet: probalo y compará el ping.",
			Risk: 1, Reboot: true, Group: "ipv6", Ops: []Op{D(tcpip6, "DisabledComponents", "32", Del)}},
		{ID: "net.noipv6", Test: true, Cat: "network", Name: "Desactivar IPv6",
			Desc: "Apaga IPv6 por completo. Útil si tu red lo maneja mal.",
			Warn: "Algunos servicios y redes solo IPv6 dejarían de funcionar. Es excluyente con «Preferir IPv4».",
			Risk: 1, Reboot: true, Group: "ipv6", Ops: []Op{D(tcpip6, "DisabledComponents", "255", Del)}},
		{ID: "net.noteredo", Test: true, Cat: "network", Name: "Desactivar Teredo",
			Desc: "Teredo es un túnel IPv6 sobre IPv4 que casi nadie usa y que a veces agrega latencia al conectarse a juegos.",
			Warn: "Algunos juegos de Xbox y apps de la Microsoft Store usan Teredo para el multijugador entre pares.",
			Risk: 1, Ops: []Op{
				Cmd("netsh.exe", "interface", "teredo", "set", "state", "disabled").Rev("interface", "teredo", "set", "state", "default"),
			}},

		// ---- Privacidad y limpieza ----
		{ID: "pr.wpbt", Cat: "privacy", Name: "Desactivar WPBT (programas del firmware al arrancar)",
			Desc: "Impide que el firmware de la placa instale y ejecute programas en Windows en cada arranque (el mecanismo que usan algunos fabricantes para sus utilidades).",
			Risk: 0, Rec: true, Reboot: true, Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager`, "DisableWpbtExecution", "1", Del),
			}},
		{ID: "pr.devmeta", Cat: "privacy", Name: "No descargar apps acompañantes de dispositivos",
			Desc: "Windows deja de buscar en internet información y apps del fabricante cuando conectás un periférico.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Device Metadata`, "PreventDeviceMetadataFromNetwork", "1", Del),
			}},
		{ID: "db.razer", Cat: "debloat", Name: "Bloquear la instalación automática de software Razer",
			Desc: "Cuando conectás un periférico Razer, Windows no instala solo Razer Synapse ni su servicio.",
			Warn: "Si usás Razer Synapse, instalalo a mano desde su sitio: este ajuste solo frena la instalación automática.",
			Risk: 1, Ops: []Op{denyDirOps(`(Join-Path $env:SystemRoot 'Installer\Razer')`, "")}},
		{ID: "db.logi", Cat: "debloat", Name: "Bloquear Logitech Download Assistant",
			Desc: "Detiene y bloquea el asistente de descargas de Logitech que se ejecuta en cada inicio de sesión.",
			Warn: "No afecta a G HUB ni a los drivers ya instalados.",
			Risk: 1, Ops: []Op{denyDirOps(`(Join-Path $env:ProgramFiles 'LogiDownloadAssistant')`, "LogiDownloadAssistant")}},
		{ID: "ui.endtask", Cat: "ui", Name: "Terminar tarea con clic derecho en la barra de tareas", Cond: "win11",
			Desc: "Agrega «Finalizar tarea» al menú de las apps de la barra: cerrás un juego colgado sin abrir el Administrador de tareas.",
			Risk: 0, Rec: true, Ops: []Op{D(explAdv+`\TaskbarDeveloperSettings`, "TaskbarEndTask", "1", Del)}},
		{ID: "ui.lockscreen", Cat: "ui", Name: "Desactivar la pantalla de bloqueo",
			Desc: "Al encender o desbloquear vas directo al inicio de sesión, sin la pantalla con foto y reloj.",
			Risk: 0, Ops: []Op{D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Personalization`, "NoLockScreen", "1", Del)}},
		{ID: "ui.notifs", Cat: "ui", Name: "Desactivar notificaciones y centro de calendario",
			Desc: "Sin avisos emergentes de Windows ni del centro de notificaciones mientras jugás.",
			Warn: "No te enterarás de avisos de apps (correo, mensajes) hasta volver a activarlo.",
			Risk: 1, Ops: []Op{
				D(`HKCU\Software\Policies\Microsoft\Windows\Explorer`, "DisableNotificationCenter", "1", Del),
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\PushNotifications`, "ToastEnabled", "0", "1"),
			}},

		// ---- Estética y comodidad ----
		{ID: "ux.fileext", Cat: "comfort", Name: "Mostrar las extensiones de archivo",
			Desc: "Ves el .exe, .zip o .jpg de cada archivo. Ayuda a no abrir un archivo disfrazado.",
			Risk: 0, Ops: []Op{D(explAdv, "HideFileExt", "0", "1")}},
		{ID: "ux.hidden", Cat: "comfort", Name: "Mostrar archivos y carpetas ocultos",
			Desc: "El Explorador deja de esconder carpetas como AppData.",
			Risk: 0, Ops: []Op{D(explAdv, "Hidden", "1", "2")}},
		{ID: "ux.dark", Cat: "comfort", Name: "Tema oscuro de Windows y de las apps",
			Desc: "Cambia el modo de color de Windows y de las apps compatibles a oscuro.",
			Risk: 0, Ops: []Op{D(personalize, "AppsUseLightTheme", "0", "1"), D(personalize, "SystemUsesLightTheme", "0", "1")}},
		{ID: "ux.numlock", Cat: "comfort", Name: "Bloq Num activado al iniciar",
			Desc: "El teclado numérico queda encendido al entrar a Windows.",
			Risk: 0, Ops: []Op{S(`HKCU\Control Panel\Keyboard`, "InitialKeyboardIndicators", "2", "0")}},
		{ID: "ux.scrollbars", Cat: "comfort", Name: "Barras de desplazamiento siempre visibles",
			Desc: "Las barras no se esconden cuando no las usás.",
			Risk: 0, Ops: []Op{D(`HKCU\Control Panel\Accessibility`, "DynamicScrollbars", "0", "1")}},
		{ID: "ux.longpaths", Cat: "comfort", Name: "Permitir rutas largas (más de 260 caracteres)",
			Desc: "Programas compatibles pueden usar rutas de archivo largas sin errores.",
			Risk: 0, Ops: []Op{D(`HKLM\SYSTEM\CurrentControlSet\Control\FileSystem`, "LongPathsEnabled", "1", "0")}},
		{ID: "ux.battery", Cat: "comfort", Name: "Porcentaje de batería en la barra", Cond: "laptop",
			Desc: "Muestra el porcentaje junto al icono de batería.",
			Risk: 0, Ops: []Op{D(explorerKey, "IsBatteryPercentageEnabled", "1", "0")}},
		{ID: "ux.bsod", Cat: "comfort", Name: "Pantalla azul con detalles técnicos",
			Desc: "Si Windows falla, la pantalla azul muestra el código del error en vez de la carita triste.",
			Risk: 0, Ops: []Op{D(crashCtl, "DisplayParameters", "1", Del), D(crashCtl, "DisableEmoticon", "1", Del)}},
		{ID: "ux.verbose", Cat: "comfort", Name: "Mensajes detallados al iniciar y apagar",
			Desc: "Al iniciar sesión y apagar, Windows muestra qué está haciendo en cada paso.",
			Risk: 0, Ops: []Op{D(`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, "VerboseStatus", "1", Del)}},
		{ID: "ux.homegallery", Cat: "comfort", Name: "Quitar «Inicio» y «Galería» del Explorador", Cond: "win11",
			Desc: "El Explorador abre en «Este equipo» y sin los accesos a Inicio y Galería.",
			Risk: 0, Ops: []Op{
				D(`HKCU\Software\Classes\CLSID\{e88865ea-0e1c-4e20-9aa6-edcd0212c87c}`, "System.IsPinnedToNameSpaceTree", "0", Del),
				D(`HKCU\Software\Classes\CLSID\{f874310e-b6b7-47dc-bc84-b9e6b38f5903}`, "System.IsPinnedToNameSpaceTree", "0", Del),
				D(explAdv, "LaunchTo", "1", Del),
			}},

		// ---- Actualizaciones ----
		{ID: "upd.pause", Cat: "updates", Watch: true, Name: "Aplazar las actualizaciones de Windows hasta el año 3000",
			Desc: "Un paréntesis en la rutina de Windows Update: pospone las actualizaciones automáticas hasta el año 3000 y detiene el orquestador que las programa. " +
				"Tu sistema conserva exactamente la configuración que elegiste, sin sorpresas ni reinicios inesperados, hasta que decidas reanudarlas con un solo clic. " +
				"Las definiciones del Antivirus de Windows no se tocan.",
			Warn: "Sin actualizaciones no recibís parches de seguridad. Windows puede reactivar el servicio tras una actualización grande: el programa lo verifica al abrirse y te avisa. " +
				"Los anti-cheat de kernel pueden exigir un Windows al día.",
			AC: "las actualizaciones de Windows (incluidos los parches de seguridad)", Risk: 2, Reboot: true, Ops: []Op{
				D(wuPol+`\AU`, "NoAutoUpdate", "1", Del),
				D(wuPol+`\AU`, "AUOptions", "1", Del),
				D(wuPol, "SetDisableUXWUAccess", "1", Del),
				S(wuUX, "PauseUpdatesStartTime", pauseStart, Del).Opt(),
				S(wuUX, "PauseUpdatesExpiryTime", pauseEnd, Del).Opt(),
				S(wuUX, "PauseFeatureUpdatesStartTime", pauseStart, Del).Opt(),
				S(wuUX, "PauseFeatureUpdatesEndTime", pauseEnd, Del).Opt(),
				S(wuUX, "PauseQualityUpdatesStartTime", pauseStart, Del).Opt(),
				S(wuUX, "PauseQualityUpdatesEndTime", pauseEnd, Del).Opt(),
				Svc("UsoSvc", 4, 3),
				Svc("WaaSMedicSvc", 4, 3).Opt(),
				PS(updTasksApply, updTasksRevert),
			}},
	}
}
