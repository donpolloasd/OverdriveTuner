package tweaks

import (
	"strings"
)

func psList(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = "'" + n + "'"
	}
	return "@(" + strings.Join(q, ",") + ")"
}

// appxOps genera los scripts para quitar (solo para el usuario actual) y
// volver a registrar paquetes. Se guarda una copia de las rutas en ProgramData.
func appxOps(id string, names []string) []Op {
	list := psList(names)
	apply := `$dir = Join-Path $env:ProgramData 'FPSForge\appx'
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$names = ` + list + `
$saved = @()
foreach ($n in $names) {
  foreach ($p in @(Get-AppxPackage -Name $n)) {
    $saved += [pscustomobject]@{ Name=$p.Name; Full=$p.PackageFullName; Loc=$p.InstallLocation }
    try { Remove-AppxPackage -Package $p.PackageFullName -ErrorAction Stop } catch { Write-Output ('No se pudo quitar ' + $p.Name + ': ' + $_.Exception.Message) }
  }
}
if ($saved.Count -gt 0) { ConvertTo-Json -InputObject @($saved) | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $dir '` + id + `.json') }
Write-Output ('Paquetes quitados: ' + $saved.Count)`
	revert := `$f = Join-Path $env:ProgramData 'FPSForge\appx\` + id + `.json'
if (-not (Test-Path -LiteralPath $f)) { Write-Output 'No hay copia guardada: reinstala desde Microsoft Store si lo necesitas.'; exit 0 }
$items = @(Get-Content -Raw -Encoding UTF8 -LiteralPath $f | ConvertFrom-Json)
foreach ($i in $items) {
  $m = Join-Path $i.Loc 'AppXManifest.xml'
  if (Test-Path -LiteralPath $m) {
    try { Add-AppxPackage -DisableDevelopmentMode -Register $m -ErrorAction Stop } catch { Write-Output ('No se pudo restaurar ' + $i.Name + ': ' + $_.Exception.Message) }
  } else { Write-Output ('Reinstala ' + $i.Name + ' desde Microsoft Store.') }
}
Remove-Item -LiteralPath $f -ErrorAction SilentlyContinue`
	return []Op{PS(apply, revert)}
}

func appTweak(id, name, desc string, rec bool, names ...string) Tweak {
	return Tweak{ID: id, Cat: "apps", Name: name, Desc: desc + " Se quita solo para tu usuario. Al revertir se intenta restaurar y, si no es posible, se reinstala desde Microsoft Store.",
		Risk: 0, Rec: rec, Ops: appxOps(id, names)}
}

func appTweaks() []Tweak {
	return []Tweak{
		appTweak("apps.msbloat", "Apps de relleno de Microsoft",
			"Noticias, Tiempo, Finanzas, Deportes, Obtener ayuda, Consejos, Office Hub, Solitario, Feedback Hub, Power Automate, To Do, Contactos, Mapas, Mixed Reality, 3D Viewer, Skype y Dev Home.",
			true, "Microsoft.BingNews", "Microsoft.BingWeather", "Microsoft.BingFinance", "Microsoft.BingSports",
			"Microsoft.GetHelp", "Microsoft.Getstarted", "Microsoft.MicrosoftOfficeHub", "Microsoft.MicrosoftSolitaireCollection",
			"Microsoft.WindowsFeedbackHub", "Microsoft.PowerAutomateDesktop", "Microsoft.Todos", "Microsoft.People",
			"Microsoft.WindowsMaps", "Microsoft.MixedReality.Portal", "Microsoft.Microsoft3DViewer", "Microsoft.Print3D",
			"Microsoft.3DBuilder", "Microsoft.OneConnect", "Microsoft.Wallet", "Microsoft.SkypeApp", "Microsoft.Windows.DevHome"),
		appTweak("apps.thirdparty", "Apps promocionadas de terceros",
			"Candy Crush, Disney+, TikTok, Facebook, Instagram, Twitter/X, LinkedIn, Netflix, Prime Video, Duolingo, Hulu y Flipboard.",
			true, "*CandyCrush*", "*Disney*", "*TikTok*", "*Facebook*", "*Instagram*", "*Twitter*", "*LinkedIn*",
			"*Netflix*", "*AmazonVideo.PrimeVideo*", "*Duolingo*", "*Hulu*", "*Flipboard*"),
		appTweak("apps.clipchamp", "Clipchamp", "Editor de video preinstalado.", true, "Clipchamp.Clipchamp"),
		appTweak("apps.teams", "Microsoft Teams personal (Chat)", "La versión de consumo de Teams integrada en Windows 11.", true, "MicrosoftTeams"),
		appTweak("apps.cortana", "Cortana", "Asistente de voz heredado.", true, "Microsoft.549981C3F5F10"),
		appTweak("apps.copilot", "Copilot", "App de Copilot y su proveedor.", false, "Microsoft.Copilot", "Microsoft.Windows.Ai.Copilot.Provider"),
		appTweak("apps.phone", "Vínculo con el móvil (Phone Link)", "Conexión con Android/iPhone.", false, "Microsoft.YourPhone", "MicrosoftWindows.CrossDevice"),
		appTweak("apps.mail", "Correo, Calendario y Outlook nuevo", "Apps de correo preinstaladas.", false, "microsoft.windowscommunicationsapps", "Microsoft.OutlookForWindows"),
		appTweak("apps.media", "Reproductor multimedia y Películas y TV", "Groove/Media Player y Películas y TV.", false, "Microsoft.ZuneMusic", "Microsoft.ZuneVideo"),
	}
}

// All devuelve el catálogo completo.
func All() []Tweak {
	var out []Tweak
	for _, part := range [][]Tweak{
		fpsTweaks(), latencyTweaks(), gpuTweaks(), powerTweaks(), memoryTweaks(), networkTweaks(),
		serviceTweaks(), privacyTweaks(), debloatTweaks(), appTweaks(), uiTweaks(), advancedTweaks(),
		perfTweaks11(), v21Tweaks(),
	} {
		out = append(out, part...)
	}
	for i := range out {
		if retired[out[i].ID] {
			out[i].Hidden, out[i].Rec, out[i].Comp = true, false, false
		}
	}
	return out
}

// ByID busca un tweak por identificador.
func ByID(id string) (Tweak, bool) {
	for _, t := range All() {
		if t.ID == id {
			return t, true
		}
	}
	return Tweak{}, false
}

// retired son ajustes de bajo valor: ya no se ofrecen, pero siguen existiendo para poder revertirlos
// si alguien los tenía aplicados (y para no alterar las copias de seguridad de la versión 1.0).
var retired = map[string]bool{
	"svc.spooler": true, "db.remoteassist": true, "lat.touchfeedback": true, "ui.shake": true,
	"pw.energyest": true, "mem.sleepstudy": true, "mem.8dot3": true, "mem.crashdump": true,
	"adv.spectre": true, "adv.hypervisor": true,
}
