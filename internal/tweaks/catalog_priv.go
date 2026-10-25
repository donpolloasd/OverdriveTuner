package tweaks

const (
	cdm      = `HKCU\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
	explAdv  = `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	polCloud = `HKLM\SOFTWARE\Policies\Microsoft\Windows\CloudContent`
	polSys   = `HKLM\SOFTWARE\Policies\Microsoft\Windows\System`
)

func privacyTweaks() []Tweak {
	return []Tweak{
		{ID: "pr.telemetry", Cat: "privacy", Name: "Reducir telemetría de Windows al mínimo",
			Desc: "Fija AllowTelemetry en 0 y desactiva las solicitudes de opinión de Microsoft. (Home/Pro aplican el mínimo permitido.)",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry", "0", Del),
				D(`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\DataCollection`, "AllowTelemetry", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "DoNotShowFeedbackNotifications", "1", Del),
				D(`HKCU\Software\Microsoft\Siuf\Rules`, "NumberOfSIUFInPeriod", "0", Del),
			}},
		{ID: "pr.appcompat", Cat: "privacy", Name: "Desactivar inventario de apps y grabador de pasos",
			Desc: "Apaga el programa de mejora de compatibilidad (AIT), el inventario y el grabador de acciones del usuario.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AppCompat`, "AITEnable", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AppCompat`, "DisableInventory", "1", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AppCompat`, "DisableUAR", "1", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AppCompat`, "DisablePCA", "1", Del),
			}},
		{ID: "pr.ads", Cat: "privacy", Name: "Desactivar ID de publicidad",
			Desc: "Las apps no pueden usar tu identificador publicitario ni el idioma para segmentar anuncios.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo`, "Enabled", "0", "1"),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo`, "DisabledByGroupPolicy", "1", Del),
				D(`HKCU\Control Panel\International\User Profile`, "HttpAcceptLanguageOptOut", "1", Del),
			}},
		{ID: "pr.tailored", Cat: "privacy", Name: "Desactivar experiencias personalizadas con datos de diagnóstico",
			Desc: "Windows no usa tus datos de diagnóstico para mostrarte sugerencias y anuncios.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Privacy`, "TailoredExperiencesWithDiagnosticDataEnabled", "0", Del),
				D(`HKCU\Software\Policies\Microsoft\Windows\CloudContent`, "DisableTailoredExperiencesWithDiagnosticData", "1", Del),
			}},
		{ID: "pr.activity", Cat: "privacy", Name: "Desactivar historial de actividad",
			Desc: "Windows deja de registrar y subir tu actividad (Timeline).",
			Risk: 0, Rec: true, Ops: []Op{
				D(polSys, "EnableActivityFeed", "0", Del),
				D(polSys, "PublishUserActivities", "0", Del),
				D(polSys, "UploadUserActivities", "0", Del),
			}},
		{ID: "pr.inking", Cat: "privacy", Name: "Desactivar recopilación de escritura y voz",
			Desc: "No se recopilan patrones de escritura, contactos ni voz para 'mejorar' el reconocimiento.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\InputPersonalization`, "RestrictImplicitInkCollection", "1", Del),
				D(`HKCU\Software\Microsoft\InputPersonalization`, "RestrictImplicitTextCollection", "1", Del),
				D(`HKCU\Software\Microsoft\InputPersonalization\TrainedDataStore`, "HarvestContacts", "0", Del),
				D(`HKCU\Software\Microsoft\Personalization\Settings`, "AcceptedPrivacyPolicy", "0", Del),
				D(`HKCU\Software\Microsoft\Speech_OneCore\Settings\OnlineSpeechPrivacy`, "HasAccepted", "0", Del),
			}},
		{ID: "pr.location", Cat: "privacy", Name: "Desactivar ubicación del sistema",
			Desc: "Bloquea el acceso a la ubicación para Windows y las apps.",
			Warn: "Se pierde 'Encontrar mi dispositivo' y la zona horaria automática por ubicación.",
			Risk: 1, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\LocationAndSensors`, "DisableLocation", "1", Del),
			}},
		{ID: "pr.clipsync", Cat: "privacy", Name: "Desactivar sincronización del portapapeles en la nube",
			Desc: "El historial (Win+V) sigue funcionando, pero no se envía a tu cuenta Microsoft.",
			Risk: 0, Ops: []Op{
				D(polSys, "AllowCrossDeviceClipboard", "0", Del),
			}},
	}
}

func debloatTweaks() []Tweak {
	cdmOps := []Op{}
	for _, n := range []string{"ContentDeliveryAllowed", "OemPreInstalledAppsEnabled", "PreInstalledAppsEnabled",
		"PreInstalledAppsEverEnabled", "SilentInstalledAppsEnabled", "SystemPaneSuggestionsEnabled", "SoftLandingEnabled",
		"FeatureManagementEnabled", "RotatingLockScreenEnabled", "RotatingLockScreenOverlayEnabled",
		"SubscribedContent-310093Enabled", "SubscribedContent-338387Enabled", "SubscribedContent-338388Enabled",
		"SubscribedContent-338389Enabled", "SubscribedContent-338393Enabled", "SubscribedContent-353694Enabled",
		"SubscribedContent-353696Enabled", "SubscribedContent-353698Enabled"} {
		cdmOps = append(cdmOps, D(cdm, n, "0", Del))
	}
	return []Tweak{
		{ID: "db.suggestions", Cat: "debloat", Name: "Quitar sugerencias, anuncios y apps promocionadas",
			Desc: "Apaga el Content Delivery Manager: sin anuncios en Inicio, pantalla de bloqueo y Configuración, y sin instalación silenciosa de apps.",
			Risk: 0, Rec: true, Ops: cdmOps},
		{ID: "db.consumer", Cat: "debloat", Name: "Bloquear 'experiencias de consumo' de Microsoft",
			Desc: "Políticas que impiden que Windows reinstale apps promocionales y muestre sugerencias de terceros.",
			Risk: 0, Rec: true, Ops: []Op{
				D(polCloud, "DisableWindowsConsumerFeatures", "1", Del),
				D(polCloud, "DisableSoftLanding", "1", Del),
				D(polCloud, "DisableCloudOptimizedContent", "1", Del),
				D(polCloud, "DisableWindowsSpotlightFeatures", "1", Del),
				D(polCloud, "DisableThirdPartySuggestions", "1", Del),
			}},
		{ID: "db.welcome", Cat: "debloat", Name: "Quitar pantallas de bienvenida y 'terminar de configurar'",
			Desc: "Elimina las pantallas tras actualizar y los avisos para terminar de configurar el equipo.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\UserProfileEngagement`, "ScoobeSystemSettingEnabled", "0", Del),
			}},
		{ID: "db.widgets", Cat: "debloat", Name: "Desactivar Widgets (Noticias e intereses)",
			Desc: "Desactiva el panel de Widgets de Windows 11 y su proceso en segundo plano.",
			Risk: 0, Rec: true, Cond: "win11", Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", "0", Del),
				D(explAdv, "TaskbarDa", "0", "1").Opt(),
			}},
		{ID: "db.chat", Cat: "debloat", Name: "Quitar Chat / Teams de la barra de tareas",
			Desc: "Oculta el icono de Chat (Teams personal) y evita que se inicie.",
			Risk: 0, Rec: true, Cond: "win11", Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Chat`, "ChatIcon", "3", Del),
				D(explAdv, "TaskbarMn", "0", "1").Opt(),
			}},
		{ID: "db.copilot", Cat: "debloat", Name: "Desactivar Copilot",
			Desc: "Aplica la política que deshabilita Copilot y oculta su botón.",
			Risk: 0, Rec: true, Cond: "win11", Ops: []Op{
				D(`HKCU\Software\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot", "1", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot", "1", Del),
				D(explAdv, "ShowCopilotButton", "0", Del).Opt(),
			}},
		{ID: "db.recall", Cat: "debloat", Name: "Desactivar Recall y análisis de datos con IA",
			Desc: "Bloquea la captura y el análisis de IA de tu actividad (Recall) en equipos que lo incluyan.",
			Risk: 0, Rec: true, Cond: "win11", Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsAI`, "DisableAIDataAnalysis", "1", Del),
			}},
		{ID: "db.bing", Cat: "debloat", Name: "Quitar Bing y la web de la búsqueda de Inicio",
			Desc: "La búsqueda de Windows muestra solo resultados locales, sin sugerencias web ni Cortana.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Policies\Microsoft\Windows\Explorer`, "DisableSearchBoxSuggestions", "1", Del),
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Search`, "BingSearchEnabled", "0", Del),
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Search`, "CortanaConsent", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "AllowCortana", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "DisableWebSearch", "1", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "ConnectedSearchUseWeb", "0", Del),
			}},
		{ID: "db.highlights", Cat: "debloat", Name: "Quitar 'Destacados de búsqueda'",
			Desc: "Elimina los dibujos y noticias del cuadro de búsqueda.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\SearchSettings`, "IsDynamicSearchBoxEnabled", "0", "1"),
			}},
		{ID: "db.startmenu", Cat: "debloat", Name: "Limpiar menú Inicio (recomendados y promos)",
			Desc: "Oculta archivos recientes, sugerencias y avisos de cuenta en el menú Inicio.",
			Risk: 0, Rec: true, Ops: []Op{
				D(explAdv, "Start_TrackDocs", "0", "1"),
				D(explAdv, "Start_IrisRecommendations", "0", "1"),
				D(explAdv, "Start_AccountNotifications", "0", "1"),
			}},
		{ID: "db.edge", Cat: "debloat", Name: "Edge: sin arranque anticipado ni segundo plano",
			Desc: "Evita que Edge se precargue y siga ejecutándose tras cerrarlo; quita recomendaciones y asistente de compras.",
			Warn: "Edge mostrará 'administrado por tu organización' porque se usan políticas.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Edge`, "StartupBoostEnabled", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Edge`, "BackgroundModeEnabled", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Edge`, "EdgeShoppingAssistantEnabled", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Edge`, "ShowRecommendationsEnabled", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Edge`, "PersonalizationReportingEnabled", "0", Del),
			}},
		{ID: "db.onedrive", Cat: "debloat", Name: "Desactivar sincronización de OneDrive",
			Desc: "Impide que OneDrive se inicie y sincronice archivos.",
			Warn: "Dejarás de sincronizar tus carpetas con la nube hasta revertir.",
			Risk: 1, Ops: []Op{
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\OneDrive`, "DisableFileSyncNGSC", "1", Del),
			}},
		{ID: "db.wudrivers", Cat: "debloat", Name: "Que Windows Update no reemplace tus drivers",
			Desc: "Excluye drivers de las actualizaciones de calidad: tu driver de GPU (Adrenalin/GeForce) no será sobrescrito.",
			Risk: 0, Rec: true, Comp: true, Ops: []Op{
				D(wuPol, "ExcludeWUDriversInQualityUpdate", "1", Del),
			}},
		{ID: "db.wureboot", Cat: "debloat", Name: "No reiniciar solo con sesión iniciada",
			Desc: "Windows Update ya no reinicia el equipo automáticamente mientras estás usándolo.",
			Risk: 0, Rec: true, Ops: []Op{
				D(wuPol+`\AU`, "NoAutoRebootWithLoggedOnUsers", "1", Del),
			}},
		{ID: "db.remoteassist", Cat: "debloat", Name: "Desactivar Asistencia remota",
			Desc: "Cierra la puerta a conexiones de asistencia remota entrantes.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\Remote Assistance`, "fAllowToGetHelp", "0", "1"),
			}},
		{ID: "db.contextmenu", Cat: "debloat", Name: "Menú contextual clásico de Windows 10",
			Desc: "Recupera el menú completo de clic derecho sin 'Mostrar más opciones'. Cierra sesión para verlo.",
			Risk: 0, Cond: "win11", Ops: []Op{
				S(`HKCU\Software\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\InprocServer32`, "", "", Del).
					DelKey(`HKCU\Software\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}`),
			}},
		{ID: "db.taskview", Cat: "debloat", Name: "Quitar botón Vista de tareas y cuadro de búsqueda de la barra",
			Desc: "Barra de tareas más limpia.",
			Risk: 0, Ops: []Op{
				D(explAdv, "ShowTaskViewButton", "0", "1"),
				D(`HKCU\Software\Microsoft\Windows\CurrentVersion\Search`, "SearchboxTaskbarMode", "0", "1"),
			}},
	}
}

func uiTweaks() []Tweak {
	return []Tweak{
		{ID: "ui.anim", Cat: "ui", Name: "Desactivar animaciones de ventanas y barra de tareas",
			Desc: "Minimizar/maximizar al instante, sin sombras de lista ni Aero Peek.",
			Risk: 0, Comp: true, Ops: []Op{
				S(`HKCU\Control Panel\Desktop\WindowMetrics`, "MinAnimate", "0", "1"),
				D(explAdv, "TaskbarAnimations", "0", "1"),
				D(explAdv, "ListviewAlphaSelect", "0", "1"),
				D(explAdv, "ListviewShadow", "0", "1"),
				D(`HKCU\Software\Microsoft\Windows\DWM`, "EnableAeroPeek", "0", "1"),
			}},
		{ID: "ui.menudelay", Cat: "ui", Name: "Menús sin retraso",
			Desc: "Los menús se abren al instante (MenuShowDelay = 0).",
			Risk: 0, Rec: true, Ops: []Op{
				S(`HKCU\Control Panel\Desktop`, "MenuShowDelay", "0", "400"),
			}},
		{ID: "ui.shake", Cat: "ui", Name: "Desactivar 'agitar para minimizar'",
			Desc: "Evita minimizar ventanas sin querer al sacudir una.",
			Risk: 0, Ops: []Op{
				D(explAdv, "DisallowShaking", "1", Del),
			}},
		{ID: "ui.killapp", Cat: "ui", Name: "Cerrar apps colgadas más rápido",
			Desc: "Reduce a 2 s el tiempo que Windows espera antes de cerrar una app que no responde o al apagar.",
			Warn: "Una app que tarde en guardar al apagar puede perder datos sin guardar.",
			Risk: 1, Ops: []Op{
				S(`HKCU\Control Panel\Desktop`, "HungAppTimeout", "2000", "5000"),
				S(`HKCU\Control Panel\Desktop`, "WaitToKillAppTimeout", "2000", "20000"),
				S(`HKLM\SYSTEM\CurrentControlSet\Control`, "WaitToKillServiceTimeout", "2000", "5000"),
			}},
	}
}

func advancedTweaks() []Tweak {
	return []Tweak{
		{ID: "adv.vbs", Cat: "advanced", Name: "Desactivar VBS / Integridad de memoria (HVCI)",
			Desc: "Quita la virtualización de seguridad de Windows 11: puede dar entre 5% y 10% más de FPS en CPU modestas.",
			Warn: "REDUCE LA SEGURIDAD: deja el núcleo sin protección contra drivers maliciosos. Requiere reiniciar.",
			Risk: 2, Reboot: true, Cond: "win11", AC: "Integridad de memoria (HVCI)", Ops: []Op{
				D(`HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity`, "Enabled", "0", Del),
				D(`HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard`, "EnableVirtualizationBasedSecurity", "0", Del),
			}},
		{ID: "adv.spectre", Cat: "advanced", Name: "Desactivar mitigaciones Spectre/Meltdown",
			Desc: "Quita parches de seguridad de CPU. La ganancia es pequeña en CPU modernas.",
			Warn: "REDUCE LA SEGURIDAD: expone el equipo a ataques de canal lateral. No recomendado. Requiere reiniciar.",
			Risk: 2, Reboot: true, Ops: []Op{
				D(memMgmt, "FeatureSettingsOverride", "3", Del),
				D(memMgmt, "FeatureSettingsOverrideMask", "3", Del),
			}},
		{ID: "adv.hypervisor", Cat: "advanced", Name: "Desactivar el hipervisor de Windows",
			Desc: "Apaga Hyper-V en el arranque: menor latencia en algunos juegos.",
			Warn: "Rompe WSL2, Docker, Windows Sandbox y emuladores Android. Requiere reiniciar.",
			Risk: 2, Reboot: true, AC: "Integridad de memoria (HVCI) y virtualización", Ops: []Op{
				Cmd("bcdedit.exe", "/set", "hypervisorlaunchtype", "off").Rev("/set", "hypervisorlaunchtype", "auto"),
			}},
	}
}
