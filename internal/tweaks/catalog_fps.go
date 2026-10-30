package tweaks

const (
	gameBar   = `HKCU\Software\Microsoft\GameBar`
	gameCfg   = `HKCU\System\GameConfigStore`
	gameDVR   = `HKCU\Software\Microsoft\Windows\CurrentVersion\GameDVR`
	sysProf   = `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	gamesTask = sysProf + `\Tasks\Games`
	mouseKey  = `HKCU\Control Panel\Mouse`
	accessKey = `HKCU\Control Panel\Accessibility`
	gfxDrv    = `HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers`
)

func fpsTweaks() []Tweak {
	return []Tweak{
		{ID: "fps.gamemode", Cat: "fps", Name: "Activar Modo Juego de Windows",
			Desc: "Prioriza el juego en primer plano y evita que Windows Update instale drivers o reinicie mientras juegas.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(gameBar, "AutoGameModeEnabled", "1", Del),
				D(gameBar, "AllowAutoGameMode", "1", Del),
			}},
		{ID: "fps.gamedvr", Cat: "fps", Name: "Desactivar Game DVR (grabación en segundo plano)",
			Desc: "Evita que Windows grabe clips en segundo plano: menos uso de GPU/CPU y menos microcortes.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(gameCfg, "GameDVR_Enabled", "0", Del),
				D(gameDVR, "AppCaptureEnabled", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\GameDVR`, "AllowGameDVR", "0", Del),
			}},
		{ID: "fps.gamebar", Cat: "fps", Name: "Desactivar el overlay de Xbox Game Bar",
			Desc: "Quita el panel de Game Bar y su aviso de inicio. Los atajos Win+G dejan de abrirlo.",
			Risk: 0, Ops: []Op{
				D(gameBar, "UseNexusForGameBarEnabled", "0", Del),
				D(gameBar, "ShowStartupPanel", "0", Del),
			}},
		{ID: "fps.fso", Test: true, Cat: "fps", Name: "Desactivar optimizaciones de pantalla completa (global)",
			Desc: "Usa pantalla completa exclusiva clásica: puede bajar latencia en juegos DX9/DX11.",
			Warn: "Algunos juegos DX12 funcionan mejor con las optimizaciones activadas; si notas problemas, revierte. Con este ajuste los juegos en pantalla completa pasan a pantalla completa exclusiva y el overlay de OverdriveTuner no se ve encima (Windows no lo permite); en «ventana sin bordes» sí se ve.",
			Risk: 1, Comp: true, Ops: []Op{
				D(gameCfg, "GameDVR_FSEBehaviorMode", "2", Del),
				D(gameCfg, "GameDVR_HonorUserFSEBehaviorMode", "1", Del),
				D(gameCfg, "GameDVR_FSEBehavior", "2", Del),
				D(gameCfg, "GameDVR_DXGIHonorFSEWindowsCompatible", "1", Del),
			}},
		{ID: "fps.windowed", Test: true, Cat: "fps", Name: "Optimizaciones para juegos en ventana (VRR + flip)",
			Desc: "Activa Variable Refresh Rate y el modelo flip en juegos DX10/11 en ventana o sin bordes. Sobrescribe esa clave; al revertir se restaura el valor anterior.",
			Risk: 0, Rec: true, Comp: true, Cond: "win11", Ops: []Op{
				S(`HKCU\Software\Microsoft\DirectX\UserGpuPreferences`, "DirectXUserGlobalSettings", "VRROptimizeEnable=1;SwapEffectUpgradeEnable=1;", Del),
			}},
		{ID: "fps.mmcss", Test: true, Cat: "fps", Name: "Prioridad máxima de CPU/GPU para juegos (MMCSS)",
			Desc: "Sube la prioridad del planificador multimedia para la categoría 'Games'.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(gamesTask, "GPU Priority", "8", "8"),
				D(gamesTask, "Priority", "6", "2"),
				S(gamesTask, "Scheduling Category", "High", "Medium"),
				S(gamesTask, "SFIO Priority", "High", "Normal"),
			}},
		{ID: "fps.sysresp", Cat: "fps", Name: "Menos CPU reservada a tareas en segundo plano",
			Desc: "Reduce la reserva de CPU de MMCSS (SystemResponsiveness) de 20% a 10%.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(sysProf, "SystemResponsiveness", "10", "20"),
			}},
		{ID: "fps.netthrottle", Cat: "fps", Name: "Quitar el limitador de red multimedia",
			Desc: "Desactiva el throttling de paquetes de red que Windows aplica mientras se reproduce multimedia.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(sysProf, "NetworkThrottlingIndex", "4294967295", "10"),
			}},
		{ID: "fps.priosep", Test: true, Cat: "fps", Name: "Prioridad de primer plano optimizada (0x26)",
			Desc: "Win32PrioritySeparation=0x26: quantums cortos y variables con refuerzo 3:1 para la app activa.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\PriorityControl`, "Win32PrioritySeparation", "38", "2"),
			}},
		{ID: "fps.powerthrottle", Cat: "fps", Name: "Desactivar Power Throttling",
			Desc: "Impide que Windows limite procesos en segundo plano para ahorrar energía (afecta launchers, voz, OBS).",
			Risk: 0, Rec: true, Comp: true, Cond: "desktop", Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff", "1", Del),
			}},
		{ID: "fps.bgapps", Cat: "fps", Name: "Impedir apps en segundo plano",
			Desc: "Evita que las apps de la Tienda corran en segundo plano y consuman CPU/RAM/red.",
			Warn: "Algunas apps de la Tienda dejarán de mostrar notificaciones con la app cerrada.",
			Risk: 1, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, "GlobalUserDisabled", "1", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AppPrivacy`, "LetAppsRunInBackground", "2", Del),
			}},
		{ID: "fps.timer", Test: true, Cat: "fps", Name: "Resolución de timer global (alta precisión)",
			Desc: "Restaura el comportamiento global del temporizador de 0,5 ms en Windows 11: frametimes más consistentes en algunos juegos.",
			Warn: "Puede aumentar levemente el consumo de energía. Requiere reiniciar.",
			Risk: 1, Comp: true, Reboot: true, Cond: "win11", Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\kernel`, "GlobalTimerResolutionRequests", "1", Del),
			}},
	}
}

func latencyTweaks() []Tweak {
	return []Tweak{
		{ID: "lat.mouseaccel", Cat: "latency", Name: "Desactivar aceleración del mouse",
			Desc: "Movimiento 1:1 del puntero (sin 'Mejorar la precisión del puntero'): puntería más consistente.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				S(mouseKey, "MouseSpeed", "0", "1"),
				S(mouseKey, "MouseThreshold1", "0", "6"),
				S(mouseKey, "MouseThreshold2", "0", "10"),
				Cmd("rundll32.exe", "user32.dll,UpdatePerUserSystemParameters").Rev("user32.dll,UpdatePerUserSystemParameters").Helper(),
			}},
		{ID: "lat.sticky", Cat: "latency", Name: "Quitar avisos de Teclas especiales",
			Desc: "Evita el aviso de Sticky/Filter/Toggle Keys al pulsar Shift 5 veces en pleno juego.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				S(accessKey+`\StickyKeys`, "Flags", "506", "510"),
				S(accessKey+`\ToggleKeys`, "Flags", "58", "62"),
				S(accessKey+`\Keyboard Response`, "Flags", "122", "126"),
			}},
		{ID: "lat.kbd", Cat: "latency", Name: "Teclado: repetición sin retraso",
			Desc: "Retraso de repetición mínimo y velocidad máxima de repetición de teclas.",
			Risk: 0, Comp: true, Ops: []Op{
				S(`HKCU\Control Panel\Keyboard`, "KeyboardDelay", "0", "1"),
				S(`HKCU\Control Panel\Keyboard`, "KeyboardSpeed", "31", "31"),
			}},
		{ID: "lat.dyntick", Test: true, Cat: "latency", Name: "Desactivar Dynamic Tick",
			Desc: "El reloj del sistema deja de 'dormir' entre ticks: reduce jitter de temporización.",
			Warn: "Efecto variable según el hardware; algo más de consumo en reposo. Requiere reiniciar.",
			Risk: 1, Comp: true, Reboot: true, Ops: []Op{
				Cmd("bcdedit.exe", "/set", "disabledynamictick", "yes").Rev("/deletevalue", "disabledynamictick"),
			}},
		{ID: "lat.hid", Cat: "latency", Name: "Reducir retraso al abrir Explorer y apps de inicio",
			Desc: "Elimina el retraso artificial al iniciar programas de arranque tras iniciar sesión.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Serialize`, "StartupDelayInMSec", "0", Del),
			}},
	}
}

func gpuTweaks() []Tweak {
	return []Tweak{
		{ID: "gpu.hags", Test: true, Cat: "gpu", Name: "Programación de GPU acelerada por hardware (HAGS)",
			Desc: "Activa HAGS: necesaria para Frame Generation y puede reducir la latencia de CPU en GPUs recientes.",
			Warn: "Requiere GPU y driver compatibles. Requiere reiniciar.",
			Risk: 1, Reboot: true, Cond: "win11", Ops: []Op{
				D(gfxDrv, "HwSchMode", "2", Del),
			}},
		{ID: "gpu.mpo", Test: true, Cat: "gpu", Name: "Desactivar Multiplane Overlay (MPO)",
			Desc: "Soluciona parpadeos, pantallas negras y stutters en algunas combinaciones de GPU/monitor.",
			Warn: "Solo útil si tienes esos síntomas. Requiere reiniciar.",
			Risk: 1, Reboot: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Microsoft\Windows\Dwm`, "OverlayTestMode", "5", Del),
			}},
		{ID: "gpu.tdr", Cat: "gpu", Name: "Tiempo de espera TDR más tolerante",
			Desc: "Evita reinicios del driver gráfico ('El controlador dejó de responder') en cargas pesadas.",
			Risk: 0, Reboot: true, Ops: []Op{
				D(gfxDrv, "TdrDelay", "10", Del),
				D(gfxDrv, "TdrDdiDelay", "25", Del),
			}},
		{ID: "gpu.nvtelemetry", Cat: "gpu", Name: "NVIDIA: desactivar telemetría",
			Desc: "Detiene el contenedor de telemetría de NVIDIA y sus tareas programadas.",
			Risk: 0, Rec: true, Cond: "nvidia", Ops: []Op{
				Svc("NvTelemetryContainer", 4, 2),
				PS(`Get-ScheduledTask -TaskName 'NvTmRep*','NvTmMon*','NvDriverUpdateCheckDaily*' -ErrorAction SilentlyContinue | Disable-ScheduledTask -ErrorAction SilentlyContinue | Out-Null`,
					`Get-ScheduledTask -TaskName 'NvTmRep*','NvTmMon*','NvDriverUpdateCheckDaily*' -ErrorAction SilentlyContinue | Enable-ScheduledTask -ErrorAction SilentlyContinue | Out-Null`),
			}},
		{ID: "gpu.amdcrash", Cat: "gpu", Name: "AMD: desactivar Crash Defender (telemetría)",
			Desc: "Detiene el servicio que recopila y envía informes de fallos del driver Radeon.",
			Risk: 0, Rec: true, Cond: "amd", Ops: []Op{
				Svc("AMD Crash Defender Service", 4, 2),
			}},
		{ID: "gpu.transp", Cat: "gpu", Name: "Desactivar transparencias de Windows",
			Desc: "Quita el efecto acrílico/Mica: libera algo de GPU y VRAM para el juego.",
			Risk: 0, Comp: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "EnableTransparency", "0", "1"),
			}},
	}
}

func powerTweaks() []Tweak {
	return []Tweak{
		{ID: "pw.plan", Test: true, Cat: "power", Name: "Plan de energía «OverdriveTuner Gaming»",
			Desc: "Crea un plan basado en Máximo rendimiento: CPU al 100% sin aparcar núcleos, sin suspensión de USB/PCIe/disco. Al revertir vuelve tu plan anterior.",
			Warn: "Mayor consumo y temperatura en reposo. No recomendado en portátiles con batería.",
			Risk: 0, Rec: true, Comp: true, Cond: "desktop", Ops: []Op{{Kind: OpPlan}}},
		{ID: "pw.hibernate", Cat: "power", Name: "Desactivar hibernación (libera varios GB)",
			Desc: "Elimina hiberfil.sys, que ocupa gran parte de tu RAM en el disco C:.",
			Risk: 0, Rec: true, Ops: []Op{
				Cmd("powercfg.exe", "/hibernate", "off").Rev("/hibernate", "on"),
			}},
		{ID: "pw.faststart", Cat: "power", Name: "Desactivar Inicio rápido",
			Desc: "Un apagado real descarga drivers y memoria: menos fallos raros y stutters tras días de uso.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power`, "HiberbootEnabled", "0", "1"),
			}},
	}
}
