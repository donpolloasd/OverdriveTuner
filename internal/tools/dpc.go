package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// DPCResult es el resultado de la prueba de latencia del sistema: cuánto tiempo del procesador se va en
// interrupciones y DPC (el trabajo urgente de los drivers). Cuando sube, aparecen cortes de audio y
// microcortes aunque los FPS sean altos.
type DPCResult struct {
	Seconds int      `json:"seconds"`
	DPCAvg  float64  `json:"dpcAvg"` // % del procesador en DPC, promedio de todos los núcleos
	DPCMax  float64  `json:"dpcMax"` // pico del núcleo más cargado
	IntAvg  float64  `json:"intAvg"` // % en interrupciones
	IntMax  float64  `json:"intMax"`
	Core    string   `json:"core"` // núcleo con el pico
	Status  string   `json:"status"`
	Detail  string   `json:"detail"`
	Tips    []string `json:"tips,omitempty"`
}

// dpcScript mide una vez por segundo los contadores de DPC e interrupciones de cada núcleo (los mismos
// que muestra el Monitor de rendimiento; no hace falta ningún driver).
const dpcScript = `$ErrorActionPreference='SilentlyContinue'
$r=@(); for($i=0;$i -lt %d;$i++){ Start-Sleep -Milliseconds 950
 foreach($p in @(Get-CimInstance Win32_PerfFormattedData_Counters_ProcessorInformation)){ $r+=[pscustomobject]@{n=[string]$p.Name; d=[double]$p.PercentDPCTime; i=[double]$p.PercentInterruptTime} } }
ConvertTo-Json -Compress -InputObject @($r)`

// DPCTest mide durante unos segundos y evalúa el resultado.
func DPCTest(sys system.System, seconds int) (DPCResult, error) {
	out, err := sys.PowerShell(fmt.Sprintf(dpcScript, seconds), time.Duration(seconds+60)*time.Second)
	if err != nil {
		return DPCResult{}, err
	}
	var raw []struct {
		N    string
		D, I float64
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil || len(raw) == 0 {
		return DPCResult{}, fmt.Errorf("Windows no devolvió los contadores de DPC")
	}
	return judgeDPC(raw2samples(raw), seconds), nil
}

type dpcSample struct {
	Core     string
	DPC, Int float64
}

func raw2samples(raw []struct {
	N    string
	D, I float64
}) []dpcSample {
	out := make([]dpcSample, len(raw))
	for i, r := range raw {
		out[i] = dpcSample{r.N, r.D, r.I}
	}
	return out
}

func judgeDPC(ss []dpcSample, seconds int) DPCResult {
	r := DPCResult{Seconds: seconds}
	var nd, ni float64
	for _, s := range ss {
		if s.Core == "_Total" {
			r.DPCAvg += s.DPC
			r.IntAvg += s.Int
			nd++
			continue
		}
		if strings.HasSuffix(s.Core, "_Total") {
			continue // totales por grupo de procesadores
		}
		if s.DPC > r.DPCMax {
			r.DPCMax, r.Core = s.DPC, s.Core
		}
		r.IntMax = math.Max(r.IntMax, s.Int)
		ni++
	}
	if nd > 0 {
		r.DPCAvg, r.IntAvg = math.Round(r.DPCAvg/nd*100)/100, math.Round(r.IntAvg/nd*100)/100
	}
	r.DPCMax, r.IntMax = math.Round(r.DPCMax*10)/10, math.Round(r.IntMax*10)/10
	switch {
	case r.DPCAvg >= 5 || r.DPCMax >= 30 || r.IntAvg >= 5:
		r.Status, r.Detail = "bad", "Los drivers ocupan mucho al procesador: es una causa típica de cortes de audio y microcortes."
	case r.DPCAvg >= 2 || r.DPCMax >= 12 || r.IntAvg >= 2:
		r.Status, r.Detail = "warn", "Hay picos de trabajo de drivers. Si notás cortes de audio o tirones, revisá los consejos."
	default:
		r.Status, r.Detail = "ok", "La latencia del sistema está bien: los drivers no le roban tiempo al juego."
	}
	if r.Status != "ok" {
		r.Tips = []string{
			"Actualizá los drivers de red (sobre todo Wi-Fi), audio y chipset desde la web del fabricante.",
			"Probá con cable de red en vez de Wi-Fi y desconectá dispositivos USB que no uses.",
			"Para ver qué driver exacto causa los picos, usá LatencyMon (gratis): mide driver por driver.",
		}
	}
	return r
}

// DPCDemo es un resultado de ejemplo para el modo demostración.
func DPCDemo() DPCResult {
	return judgeDPC([]dpcSample{{"_Total", 0.6, 0.3}, {"0,0", 2.1, 0.9}, {"0,1", 0.4, 0.2}}, 10)
}
