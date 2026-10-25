package telemetry

// Script es el muestreador que corre en PowerShell: una línea JSON por
// medición. Usa clases WMI de rendimiento (nombres en inglés en cualquier
// idioma de Windows) y no instala nada ni toca drivers.
const Script = `$ErrorActionPreference='SilentlyContinue'
[Console]::OutputEncoding=[Text.Encoding]::UTF8
$total=[double](Get-CimInstance Win32_OperatingSystem).TotalVisibleMemorySize
$nv=$null; if(Get-Command nvidia-smi.exe){$nv='nvidia-smi.exe'}
$n=0; $nvT=$null; $lc=$null; $lg=$null
while($true){
 $o=[ordered]@{}
 $p=@(Get-CimInstance Win32_PerfFormattedData_Counters_ProcessorInformation)
 $t=$p|Where-Object{$_.Name -eq '_Total'}|Select-Object -First 1
 $o.cpu=[double]$t.PercentProcessorUtility
 $o.mhz=[double]$t.ProcessorFrequency
 $o.max=[double](($p|Where-Object{$_.Name -notmatch '_Total'}|ForEach-Object{[double]$_.PercentProcessorUtility}|Measure-Object -Maximum).Maximum)
 $o.freeKb=[double](Get-CimInstance Win32_OperatingSystem).FreePhysicalMemory
 $o.totalKb=$total
 $g=@{}
 foreach($x in @(Get-CimInstance Win32_PerfFormattedData_GPUPerformanceCounters_GPUEngine)){
  if($x.Name -match 'luid_(0x[0-9A-Fa-f]+)_(0x[0-9A-Fa-f]+).*engtype_3D'){
   $k=($Matches[1]+'_'+$Matches[2]).ToUpper().Replace('0X','0x'); if(-not $g[$k]){$g[$k]=@{u=0.0;v=0.0}}; $g[$k].u+=[double]$x.UtilizationPercentage } }
 foreach($x in @(Get-CimInstance Win32_PerfFormattedData_GPUPerformanceCounters_GPUAdapterMemory)){
  if($x.Name -match 'luid_(0x[0-9A-Fa-f]+)_(0x[0-9A-Fa-f]+)'){
   $k=($Matches[1]+'_'+$Matches[2]).ToUpper().Replace('0X','0x'); if(-not $g[$k]){$g[$k]=@{u=0.0;v=0.0}}; $g[$k].v+=[double]$x.DedicatedUsage } }
 $o.gpu=@($g.Keys|ForEach-Object{@{luid=$_;u=[math]::Min(100,$g[$_].u);v=$g[$_].v}})
 if(($n % 3) -eq 0){
  if($nv){$nvT=$null;$r=(& $nv --query-gpu=temperature.gpu --format=csv,noheader,nounits 2>$null|Select-Object -First 1); if($r -match '^\d+'){$nvT=[double]$r}}
  $lc=$null;$lg=$null
  foreach($ns in 'root/LibreHardwareMonitor','root/OpenHardwareMonitor'){
   $s=@(Get-CimInstance -Namespace $ns -ClassName Sensor|Where-Object{$_.SensorType -eq 'Temperature'})
   if($s.Count){
    $c=$s|Where-Object{$_.Name -match 'Tctl|Tdie|CPU Package|Core Average|CPU Total'}|Select-Object -First 1; if($c){$lc=[double]$c.Value}
    $d=$s|Where-Object{$_.Name -match 'GPU Core'}|Select-Object -First 1; if($d){$lg=[double]$d.Value}
    break } } }
 $n++
 $o.nv=$nvT; $o.lhmCpu=$lc; $o.lhmGpu=$lg
 [Console]::Out.WriteLine(($o|ConvertTo-Json -Compress -Depth 4)); [Console]::Out.Flush()
 Start-Sleep -Milliseconds 500
}`
