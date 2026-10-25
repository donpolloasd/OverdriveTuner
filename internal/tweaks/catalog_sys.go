package tweaks

import "strings"

import "time"

const (
	memMgmt = `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`
	wuPol   = `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`
)

func memoryTweaks() []Tweak {
	return []Tweak{
		{ID: "mem.pagingexec", Cat: "memory", Name: "Mantener el kernel y drivers en RAM",
			Desc: "DisablePagingExecutive: el núcleo de Windows no se envía al archivo de paginación. Seguro con 16 GB o más.",
			Risk: 0, Rec: true, Comp: true, Reboot: true, Cond: "ram16", Ops: []Op{
				D(memMgmt, "DisablePagingExecutive", "1", "0"),
			}},
		{ID: "mem.memcomp", Test: true, Cat: "memory", Name: "Desactivar compresión de memoria",
			Desc: "Evita gastar CPU en comprimir RAM cuando te sobra memoria.",
			Risk: 1, Comp: true, Reboot: true, Cond: "ram16", Ops: []Op{
				PS(`Disable-MMAgent -MemoryCompression -ErrorAction Stop`, `Enable-MMAgent -MemoryCompression -ErrorAction Stop`),
			}},
		{ID: "mem.prefetch", Cat: "memory", Name: "Desactivar SysMain y Prefetch (solo SSD)",
			Desc: "En un SSD no aportan nada y generan lecturas/escrituras y uso de CPU en segundo plano.",
			Risk: 0, Rec: true, Reboot: true, Cond: "ssd", Ops: []Op{
				Svc("SysMain", 4, 2),
				D(memMgmt+`\PrefetchParameters`, "EnablePrefetcher", "0", "3"),
				D(memMgmt+`\PrefetchParameters`, "EnableSuperfetch", "0", "3"),
			}},
		{ID: "disk.lastaccess", Cat: "memory", Name: "Dejar de registrar el último acceso a archivos",
			Desc: "NTFS no escribe la hora de último acceso cada vez que lees un archivo.",
			Risk: 0, Rec: true, Ops: []Op{
				Cmd("fsutil.exe", "behavior", "set", "disablelastaccess", "1").Rev("behavior", "set", "disablelastaccess", "2"),
			}},
		{ID: "disk.maint", Cat: "memory", Name: "Desactivar mantenimiento automático",
			Desc: "Evita que Windows ejecute tareas pesadas (desfragmentación, diagnósticos) mientras juegas.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Schedule\Maintenance`, "MaintenanceDisabled", "1", Del),
			}},
		{ID: "disk.wer", Cat: "memory", Name: "Desactivar informe de errores de Windows",
			Desc: "Detiene WerSvc y el envío de informes de fallos a Microsoft.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Microsoft\Windows\Windows Error Reporting`, "Disabled", "1", Del),
				Svc("WerSvc", 4, 3),
			}},
		{ID: "disk.reserved", Cat: "memory", Name: "Liberar el almacenamiento reservado (~7 GB)",
			Desc: "Desactiva el espacio que Windows aparta para actualizaciones. Útil si tu SSD es chico.",
			Warn: "Puede fallar si hay actualizaciones pendientes; en ese caso reinicia e intenta de nuevo.",
			Risk: 1, Ops: []Op{
				Cmd("dism.exe", "/Online", "/Set-ReservedStorageState", "/State:Disabled").Rev("/Online", "/Set-ReservedStorageState", "/State:Enabled").Long(10 * time.Minute),
			}},
		{ID: "disk.indexing", Cat: "memory", Name: "Desactivar indexado de Windows Search",
			Desc: "Detiene el indexador: menos actividad de disco/CPU en segundo plano.",
			Warn: "La búsqueda de archivos en el Explorador será más lenta.",
			Risk: 1, Ops: []Op{Svc("WSearch", 4, 2)}},
	}
}

func networkTweaks() []Tweak {
	const nagleApply = `$base='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces'
Get-ChildItem -LiteralPath $base | ForEach-Object {
  New-ItemProperty -LiteralPath $_.PSPath -Name TcpAckFrequency -Value 1 -PropertyType DWord -Force | Out-Null
  New-ItemProperty -LiteralPath $_.PSPath -Name TCPNoDelay -Value 1 -PropertyType DWord -Force | Out-Null
}`
	const nagleRevert = `$base='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces'
Get-ChildItem -LiteralPath $base | ForEach-Object {
  Remove-ItemProperty -LiteralPath $_.PSPath -Name TcpAckFrequency -ErrorAction SilentlyContinue
  Remove-ItemProperty -LiteralPath $_.PSPath -Name TCPNoDelay -ErrorAction SilentlyContinue
}`
	const dnsApply = `$dir = Join-Path $env:ProgramData 'FPSForge'
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$bk = Join-Path $dir 'dns-backup.json'
$ads = @(Get-NetAdapter -Physical | Where-Object { $_.Status -eq 'Up' })
if (-not (Test-Path -LiteralPath $bk)) {
  $saved = @()
  foreach ($a in $ads) {
    $ns = ''
    try { $ns = [string](Get-ItemProperty -LiteralPath ('HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\' + $a.InterfaceGuid) -Name NameServer -ErrorAction Stop).NameServer } catch {}
    $static = @($ns -split '[,\s]+' | Where-Object { $_ })
    $saved += [pscustomobject]@{ Index=$a.ifIndex; Name=$a.Name; Servers=$static }
  }
  ConvertTo-Json -InputObject @($saved) | Set-Content -Encoding UTF8 -LiteralPath $bk
}
foreach ($a in $ads) { Set-DnsClientServerAddress -InterfaceIndex $a.ifIndex -ServerAddresses ('1.1.1.1','1.0.0.1') }
Clear-DnsClientCache`
	const dnsRevert = `$bk = Join-Path $env:ProgramData 'FPSForge\dns-backup.json'
$items = @()
if (Test-Path -LiteralPath $bk) { $items = @(Get-Content -Raw -Encoding UTF8 -LiteralPath $bk | ConvertFrom-Json) }
foreach ($i in $items) {
  $s = @($i.Servers | Where-Object { $_ })
  if ($s.Count -gt 0) { Set-DnsClientServerAddress -InterfaceIndex $i.Index -ServerAddresses $s }
  else { Set-DnsClientServerAddress -InterfaceIndex $i.Index -ResetServerAddresses }
}
if ($items.Count -eq 0) { Get-NetAdapter -Physical | ForEach-Object { Set-DnsClientServerAddress -InterfaceIndex $_.ifIndex -ResetServerAddresses } }
Remove-Item -LiteralPath $bk -ErrorAction SilentlyContinue
Clear-DnsClientCache`
	return []Tweak{
		{ID: "net.nagle", Cat: "network", Name: "Desactivar algoritmo de Nagle (TCP NoDelay)",
			Desc: "Envía paquetes TCP pequeños de inmediato en vez de agruparlos: menos latencia en juegos y chats de voz.",
			Risk: 0, Comp: true, Ops: []Op{PS(nagleApply, nagleRevert)}},
		{ID: "net.adapterpower", Cat: "network", Name: "Impedir que Windows apague el adaptador de red",
			Desc: "Evita que el adaptador se suspenda para ahorrar energía (cortes de conexión y picos de ping).",
			Risk: 0, Rec: true, Comp: true, Cond: "desktop", Ops: []Op{
				PS(`Get-NetAdapter -Physical | ForEach-Object { try { Disable-NetAdapterPowerManagement -Name $_.Name -NoRestart -ErrorAction Stop } catch {} }`,
					`Get-NetAdapter -Physical | ForEach-Object { try { Enable-NetAdapterPowerManagement -Name $_.Name -NoRestart -ErrorAction Stop } catch {} }`),
			}},
		{ID: "net.rsc", Cat: "network", Name: "Desactivar RSC (Receive Segment Coalescing)",
			Desc: "Entrega los paquetes recibidos sin agruparlos: algo menos de latencia de red.",
			Risk: 0, Comp: true, Ops: []Op{
				Cmd("netsh.exe", "int", "tcp", "set", "global", "rsc=disabled").Rev("int", "tcp", "set", "global", "rsc=enabled"),
			}},
		{ID: "net.delivopt", Cat: "network", Name: "Desactivar Delivery Optimization (P2P)",
			Desc: "Windows deja de usar tu conexión para enviar actualizaciones a otros equipos.",
			Risk: 0, Rec: true, Ops: []Op{
				D(`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\DeliveryOptimization\Config`, "DODownloadMode", "0", Del),
				D(`HKLM\SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`, "DODownloadMode", "0", Del),
			}},
		{ID: "net.dns", Cat: "network", Name: "DNS rápidos: Cloudflare 1.1.1.1",
			Desc: "Cambia el DNS de tus adaptadores activos a 1.1.1.1 / 1.0.0.1. Se guarda tu configuración anterior para restaurarla.",
			Risk: 1, Group: "dns", Ops: []Op{PS(dnsApply, dnsRevert)}},
		dnsTweak("net.dns.google", "Google 8.8.8.8", "8.8.8.8", "8.8.4.4", dnsApply, dnsRevert),
		dnsTweak("net.dns.quad9", "Quad9 9.9.9.9", "9.9.9.9", "149.112.112.112", dnsApply, dnsRevert),
		dnsTweak("net.dns.opendns", "OpenDNS 208.67.222.222", "208.67.222.222", "208.67.220.220", dnsApply, dnsRevert),
	}
}

// dnsTweak es un DNS alternativo: comparte la copia de seguridad con net.dns
// y solo puede haber uno activo a la vez (grupo «dns»).
func dnsTweak(id, name, a, b, apply, revert string) Tweak {
	return Tweak{ID: id, Cat: "network", Name: "DNS rápidos: " + name,
		Desc: "Cambia el DNS de tus adaptadores activos a " + a + " / " + b + ". Usá el test de DNS en Diagnóstico para saber cuál responde más rápido en tu conexión. Se guarda tu configuración anterior para restaurarla.",
		Risk: 1, Group: "dns", Ops: []Op{PS(strings.ReplaceAll(apply, "('1.1.1.1','1.0.0.1')", "('"+a+"','"+b+"')"), revert)}}
}

func serviceTweaks() []Tweak {
	const taskApply = `$tasks = @(
 '\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser',
 '\Microsoft\Windows\Application Experience\ProgramDataUpdater',
 '\Microsoft\Windows\Application Experience\StartupAppTask',
 '\Microsoft\Windows\Customer Experience Improvement Program\Consolidator',
 '\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip',
 '\Microsoft\Windows\Autochk\Proxy',
 '\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticDataCollector',
 '\Microsoft\Windows\Windows Error Reporting\QueueReporting')
foreach ($t in $tasks) { $p = Split-Path $t -Parent; $n = Split-Path $t -Leaf
  try { Disable-ScheduledTask -TaskPath ($p + '\') -TaskName $n -ErrorAction Stop | Out-Null } catch {} }`
	taskRevert := `$tasks = @(
 '\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser',
 '\Microsoft\Windows\Application Experience\ProgramDataUpdater',
 '\Microsoft\Windows\Application Experience\StartupAppTask',
 '\Microsoft\Windows\Customer Experience Improvement Program\Consolidator',
 '\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip',
 '\Microsoft\Windows\Autochk\Proxy',
 '\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticDataCollector',
 '\Microsoft\Windows\Windows Error Reporting\QueueReporting')
foreach ($t in $tasks) { $p = Split-Path $t -Parent; $n = Split-Path $t -Leaf
  try { Enable-ScheduledTask -TaskPath ($p + '\') -TaskName $n -ErrorAction Stop | Out-Null } catch {} }`
	return []Tweak{
		{ID: "svc.telemetry", Cat: "services", Name: "Detener servicios de telemetría",
			Desc: "Deshabilita DiagTrack, dmwappushservice y el recolector de diagnósticos.",
			Risk: 0, Rec: true, Ops: []Op{
				Svc("DiagTrack", 4, 2),
				Svc("dmwappushservice", 4, 3),
				Svc("diagnosticshub.standardcollector.service", 4, 3),
			}},
		{ID: "svc.tasks", Cat: "services", Name: "Desactivar tareas programadas de telemetría",
			Desc: "Apaga Compatibility Appraiser, CEIP, ProgramDataUpdater y otras tareas que despiertan al sistema.",
			Risk: 0, Rec: true, Ops: []Op{PS(taskApply, taskRevert)}},
		{ID: "svc.unneeded", Cat: "services", Name: "Deshabilitar servicios innecesarios",
			Desc: "Fax, demo de tienda, mapas sin conexión, seguimiento de vínculos, control parental, reproductor en red, teléfono, pagos NFC, Insider y ubicación.",
			Risk: 0, Rec: true, Ops: []Op{
				Svc("Fax", 4, 3), Svc("RetailDemo", 4, 3), Svc("MapsBroker", 4, 2), Svc("TrkWks", 4, 2),
				Svc("WpcMonSvc", 4, 3), Svc("WMPNetworkSvc", 4, 3), Svc("PhoneSvc", 4, 3), Svc("SEMgrSvc", 4, 3),
				Svc("wisvc", 4, 3), Svc("lfsvc", 4, 3), Svc("AJRouter", 4, 3),
			}},
		{ID: "svc.xbox", Cat: "services", Name: "Deshabilitar servicios de Xbox",
			Desc: "Libera RAM y procesos si no usas Xbox App, Game Pass ni juegos de la Microsoft Store.",
			Warn: "Rompe Xbox App, Game Pass, Minecraft Launcher y logros de Xbox en PC.",
			Risk: 1, Ops: []Op{
				Svc("XblAuthManager", 4, 3), Svc("XblGameSave", 4, 3), Svc("XboxGipSvc", 4, 3), Svc("XboxNetApiSvc", 4, 3),
			}},
		{ID: "svc.spooler", Cat: "services", Name: "Deshabilitar cola de impresión",
			Desc: "Detiene el servicio de impresión si no tienes impresora.",
			Warn: "No podrás imprimir ni exportar a PDF desde 'Imprimir' hasta revertir.",
			Risk: 1, Ops: []Op{Svc("Spooler", 4, 2)}},
	}
}
