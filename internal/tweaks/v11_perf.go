package tweaks

const (
	powerCtl  = `HKLM\SYSTEM\CurrentControlSet\Control\Power`
	crashCtl  = `HKLM\SYSTEM\CurrentControlSet\Control\CrashControl`
	storPol   = `HKCU\Software\Microsoft\Windows\CurrentVersion\StorageSense\Parameters\StoragePolicy`
	cursorKey = `HKCU\Control Panel\Cursors`
)

// perfTweaks11 agrega ajustes de rendimiento de la versión 1.1.
func perfTweaks11() []Tweak {
	return []Tweak{
		// ---- FPS y juegos ----
		{ID: "fps.dvrcapture", Cat: "fps", Name: "Desactivar captura de audio, cursor e historial de Game DVR",
			Desc: "Complementa «Desactivar Game DVR»: apaga la captura de audio y del cursor y el buffer histórico de clips.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(gameDVR, "HistoricalCaptureEnabled", "0", Del),
				D(gameDVR, "AudioCaptureEnabled", "0", Del),
				D(gameDVR, "CursorCaptureEnabled", "0", Del),
			}},
		{ID: "fps.gamingoverlay", Cat: "fps", Name: "Evitar el aviso «ms-gamingoverlay» al usar un mando",
			Desc: "Si desactivas Game Bar, Windows puede preguntar con qué app abrir «ms-gamingoverlay» al pulsar el botón Xbox del mando. Esto registra el protocolo para que no aparezca.",
			Risk: 0, Rec: true, Ops: []Op{
				S(`HKLM\SOFTWARE\Classes\ms-gamingoverlay`, "", "URL:ms-gamingoverlay", Del).DelKey(`HKLM\SOFTWARE\Classes\ms-gamingoverlay`),
				S(`HKLM\SOFTWARE\Classes\ms-gamingoverlay`, "URL Protocol", "", Del),
			}},
		{ID: "fps.hpet", Cat: "fps", Name: "Quitar el forzado del reloj HPET",
			Desc: "Elimina la opción de arranque «useplatformclock» si algún programa la dejó activa: forzarla aumenta la latencia del temporizador. Si no existe, no cambia nada.",
			Risk: 0, Comp: true, Reboot: true, Ops: []Op{
				Cmd("bcdedit.exe", "/deletevalue", "useplatformclock").Helper().Opt(),
			}},
		{ID: "fps.searchprio", Cat: "fps", Name: "Prioridad baja para el indexador de búsqueda",
			Desc: "SearchIndexer.exe pasa a prioridad baja de CPU y de entrada/salida, así no compite con tu juego cuando indexa.",
			Risk: 0, Rec: true, Reboot: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\SearchIndexer.exe\PerfOptions`, "CpuPriorityClass", "5", Del).
					DelKey(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\SearchIndexer.exe\PerfOptions`),
				D(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\SearchIndexer.exe\PerfOptions`, "IoPriority", "0", Del).
					DelKey(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\SearchIndexer.exe\PerfOptions`),
			}},
		{ID: "fps.defenderprio", Cat: "fps", Name: "Antivirus de Windows con prioridad baja de CPU en análisis",
			Desc: "Limita el uso máximo de CPU de los análisis programados al 20% (por defecto 50%). La protección en tiempo real no cambia.",
			Risk: 0, Rec: true, Ops: []Op{
				PS(`Set-MpPreference -ScanAvgCPULoadFactor 20 -ErrorAction Stop`, `Set-MpPreference -ScanAvgCPULoadFactor 50 -ErrorAction SilentlyContinue`),
			}},
		{ID: "fps.defendersched", Cat: "fps", Name: "Sin análisis de recuperación del antivirus al iniciar",
			Desc: "Si el PC estaba apagado a la hora del análisis rápido, Windows lo «recupera» al encender y consume CPU y disco. Esto lo evita; la protección en tiempo real sigue igual.",
			Risk: 0, Rec: true, Ops: []Op{
				PS(`Set-MpPreference -ScanOnlyIfIdleEnabled $true -DisableCatchupQuickScan $true -ErrorAction Stop`,
					`Set-MpPreference -ScanOnlyIfIdleEnabled $true -DisableCatchupQuickScan $false -ErrorAction SilentlyContinue`),
			}},

		// ---- Latencia y entrada ----
		{ID: "lat.usbsuspend", Cat: "latency", Name: "Desactivar suspensión selectiva de USB",
			Desc: "Evita que Windows apague puertos USB para ahorrar energía: mouse, teclado y mandos no «despiertan» con retraso.",
			Risk: 0, Rec: true, Comp: true, Reboot: true, Cond: "desktop", Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Services\USB`, "DisableSelectiveSuspend", "1", Del),
			}},
		{ID: "lat.touchfeedback", Cat: "latency", Name: "Quitar efectos visuales del toque y gestos",
			Desc: "Desactiva los círculos de retroalimentación táctil y de gestos del puntero.",
			Risk: 0, Ops: []Op{
				D(cursorKey, "ContactVisualization", "0", Del),
				D(cursorKey, "GestureVisualization", "0", Del),
			}},
		// ---- Gráficos ----
		{ID: "gpu.dvrfsepolicy", Test: true, Cat: "gpu", Name: "Pantalla completa exclusiva: respetar el modo clásico (DSE)",
			Desc: "Complementa a las optimizaciones de pantalla completa: fija el comportamiento DSE en el modo clásico (valor 2).",
			Warn: "Úsalo junto con «Desactivar optimizaciones de pantalla completa». Si un juego da problemas, revierte.",
			Risk: 1, Comp: true, Ops: []Op{
				D(gameCfg, "GameDVR_DSEBehavior", "2", Del),
			}},

		// ---- Energía y CPU ----
		{ID: "pw.energyest", Cat: "power", Name: "Desactivar la estimación de energía",
			Desc: "Windows deja de calcular el consumo de energía de cada app en segundo plano.",
			Risk: 0, Rec: true, Ops: []Op{
				D(powerCtl, "EnergyEstimationEnabled", "0", Del),
			}},
		// ---- Memoria y disco ----
		{ID: "mem.trim", Cat: "memory", Name: "Asegurar que TRIM esté activo (SSD)",
			Desc: "TRIM le avisa al SSD qué bloques están libres: mantiene la velocidad de escritura y alarga su vida útil.",
			Risk: 0, Rec: true, Cond: "ssd", Ops: []Op{
				Cmd("fsutil.exe", "behavior", "set", "DisableDeleteNotify", "0").Helper().Opt(),
			}},
		{ID: "mem.ntfsmem", Cat: "memory", Name: "Más memoria para la caché de NTFS",
			Desc: "Aumenta el pool paginado que usa NTFS: ayuda con muchos archivos pequeños (librerías de juegos grandes).",
			Risk: 0, Rec: true, Cond: "ram16", Reboot: true, Ops: []Op{
				Cmd("fsutil.exe", "behavior", "set", "memoryusage", "2").Rev("behavior", "set", "memoryusage", "1"),
			}},
		{ID: "mem.8dot3", Cat: "memory", Name: "Desactivar nombres cortos 8.3 en NTFS",
			Desc: "NTFS deja de generar un nombre MS-DOS para cada archivo nuevo: menos escrituras al crear archivos.",
			Warn: "Instaladores muy antiguos (16 bits) pueden fallar. Solo afecta a archivos nuevos.",
			Risk: 1, Ops: []Op{
				Cmd("fsutil.exe", "behavior", "set", "disable8dot3", "1").Rev("behavior", "set", "disable8dot3", "2"),
			}},
		{ID: "mem.crashdump", Cat: "memory", Name: "Volcado de memoria mínimo al fallar",
			Desc: "En un error grave (pantalla azul) se guarda un volcado pequeño en vez de uno de varios GB.",
			Risk: 0, Rec: true, Ops: []Op{
				D(crashCtl, "CrashDumpEnabled", "3", "7"),
			}},
		{ID: "mem.storagesense", Cat: "memory", Name: "Desactivar Sensor de almacenamiento",
			Desc: "Windows no borra por su cuenta archivos temporales ni de la papelera en segundo plano.",
			Risk: 0, Ops: []Op{
				D(storPol, "01", "0", Del),
			}},
		{ID: "mem.sleepstudy", Cat: "memory", Name: "Desactivar recopilación de datos de energía (SleepStudy)",
			Desc: "Detiene las tareas que registran consumo y suspensión del equipo.",
			Risk: 0, Rec: true, Ops: []Op{
				PS(`foreach ($n in 'SleepStudy','SleepStudyDeviceRefresh') { try { Disable-ScheduledTask -TaskPath '\Microsoft\Windows\Power Efficiency Diagnostics\' -TaskName $n -ErrorAction Stop | Out-Null } catch {} }
wevtutil sl Microsoft-Windows-SleepStudy/Diagnostic /e:false 2>$null`,
					`foreach ($n in 'SleepStudy','SleepStudyDeviceRefresh') { try { Enable-ScheduledTask -TaskPath '\Microsoft\Windows\Power Efficiency Diagnostics\' -TaskName $n -ErrorAction Stop | Out-Null } catch {} }
wevtutil sl Microsoft-Windows-SleepStudy/Diagnostic /e:true 2>$null`),
			}},
	}
}
