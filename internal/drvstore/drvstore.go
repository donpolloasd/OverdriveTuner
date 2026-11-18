// Package drvstore encuentra versiones viejas de drivers que Windows sigue guardando (el almacén de
// drivers) aunque ya se instaló una más nueva. Solo propone las que ningún dispositivo usa, ni siquiera
// uno desconectado, y las borra con pnputil sin /force: si alguna estuviera en uso, Windows se niega.
package drvstore

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// ErrUnsupported: esta versión de Windows no tiene «pnputil /format xml» (llegó con Windows 11 24H2).
var ErrUnsupported = errors.New("tu versión de Windows no permite listar los drivers de forma segura")

// Pkg es un paquete de drivers del almacén.
type Pkg struct {
	Name     string `json:"name"`     // oem12.inf
	Original string `json:"original"` // amdkmdag.inf
	Provider string `json:"provider"`
	Class    string `json:"class"`
	Date     string `json:"date"` // yyyy-mm-dd
	Version  string `json:"version"`
	SizeMB   int    `json:"sizeMB"`
	Newer    string `json:"newer"` // versión que lo reemplaza
}

type xmlDrivers struct {
	Drivers []struct {
		Name     string `xml:"DriverName,attr"`
		Original string `xml:"OriginalName"`
		Provider string `xml:"ProviderName"`
		Class    string `xml:"ClassName"`
		Version  string `xml:"DriverVersion"`
	} `xml:"Driver"`
}

type xmlDevices struct {
	Devices []struct {
		Driver string `xml:"DriverName"`
	} `xml:"Device"`
}

var oemRe = regexp.MustCompile(`(?i)^oem\d+\.inf$`)

// Old devuelve los paquetes viejos que se pueden borrar, de los que más ocupan a los que menos.
func Old(sys system.System) ([]Pkg, error) {
	drv, _, err := sys.Exec(time.Minute, "pnputil.exe", "/enum-drivers", "/format", "xml")
	if err != nil || !strings.Contains(drv, "<PnpUtil") {
		return nil, ErrUnsupported
	}
	dev, _, err := sys.Exec(time.Minute, "pnputil.exe", "/enum-devices", "/format", "xml")
	if err != nil || !strings.Contains(dev, "<PnpUtil") {
		return nil, ErrUnsupported
	}
	list, err := oldFrom(drv, dev)
	if err != nil {
		return nil, err
	}
	win := os.Getenv("WINDIR")
	for i := range list {
		list[i].SizeMB = sizeMB(win, list[i])
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].SizeMB > list[j].SizeMB })
	return list, nil
}

// oldFrom compara los drivers del almacén con los que usan los dispositivos. Un paquete es viejo si hay
// otro con el mismo nombre original y el mismo fabricante que es más nuevo, y ningún dispositivo lo usa.
func oldFrom(drvXML, devXML string) ([]Pkg, error) {
	var d xmlDrivers
	var v xmlDevices
	// pnputil dice «utf-8» pero escribe los acentos en la página de códigos de la consola: se reemplazan
	// (solo se usan nombres de archivo y versiones, que no llevan acentos).
	drvXML, devXML = strings.ToValidUTF8(drvXML, "?"), strings.ToValidUTF8(devXML, "?")
	if err := xml.Unmarshal([]byte(drvXML), &d); err != nil {
		return nil, fmt.Errorf("no se pudo leer la lista de drivers: %w", err)
	}
	if err := xml.Unmarshal([]byte(devXML), &v); err != nil {
		return nil, fmt.Errorf("no se pudo leer la lista de dispositivos: %w", err)
	}
	used := map[string]bool{}
	for _, x := range v.Devices {
		used[strings.ToLower(strings.TrimSpace(x.Driver))] = true
	}
	groups := map[string][]Pkg{}
	for _, x := range d.Drivers {
		if !oemRe.MatchString(x.Name) {
			continue
		}
		date, ver := splitVersion(x.Version)
		p := Pkg{Name: strings.ToLower(x.Name), Original: strings.ToLower(x.Original), Provider: x.Provider, Class: x.Class, Date: date, Version: ver}
		k := p.Original + "|" + strings.ToLower(p.Provider)
		groups[k] = append(groups[k], p)
	}
	var out []Pkg
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		sort.SliceStable(g, func(i, j int) bool { return newerPkg(g[i], g[j]) })
		for _, p := range g[1:] {
			if used[p.Name] || !newerPkg(g[0], p) {
				continue
			}
			p.Newer = g[0].Version
			out = append(out, p)
		}
	}
	return out, nil
}

// splitVersion separa «08/21/2025 25.30.0.1» en «2025-08-21» y «25.30.0.1».
func splitVersion(s string) (string, string) {
	f := strings.Fields(s)
	if len(f) != 2 {
		return "", strings.TrimSpace(s)
	}
	t, err := time.Parse("01/02/2006", f[0])
	if err != nil {
		return "", f[1]
	}
	return t.Format("2006-01-02"), f[1]
}

// newerPkg dice si a es más nuevo que b (por fecha y, si empatan, por versión).
func newerPkg(a, b Pkg) bool {
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	x, y := strings.Split(a.Version, "."), strings.Split(b.Version, ".")
	for i := 0; i < len(x) && i < len(y); i++ {
		p, _ := strconv.Atoi(x[i])
		q, _ := strconv.Atoi(y[i])
		if p != q {
			return p > q
		}
	}
	return len(x) > len(y)
}

// sizeMB busca la carpeta del paquete en el almacén (la que tiene el mismo .inf que Windows\INF\oemN.inf)
// y suma lo que ocupa. Devuelve 0 si no la encuentra.
func sizeMB(win string, p Pkg) int {
	if win == "" {
		return 0
	}
	inf, err := os.ReadFile(filepath.Join(win, "INF", p.Name))
	if err != nil {
		return 0
	}
	dirs, _ := filepath.Glob(filepath.Join(win, "System32", "DriverStore", "FileRepository", p.Original+"_*"))
	for _, dir := range dirs {
		if b, err := os.ReadFile(filepath.Join(dir, p.Original)); err == nil && bytes.Equal(b, inf) {
			var n int64
			_ = filepath.Walk(dir, func(_ string, fi os.FileInfo, err error) error {
				if err == nil && !fi.IsDir() {
					n += fi.Size()
				}
				return nil
			})
			return int(n / (1 << 20))
		}
	}
	return 0
}

// Delete borra un paquete viejo del almacén. Sin /force: si un dispositivo lo usa, Windows no lo borra.
func Delete(sys system.System, name string) error {
	if !oemRe.MatchString(name) {
		return fmt.Errorf("nombre de driver no válido")
	}
	out, code, err := sys.Exec(2*time.Minute, "pnputil.exe", "/delete-driver", strings.ToLower(name))
	if err != nil || code != 0 {
		msg := strings.TrimSpace(out)
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = strings.TrimSpace(msg[i+1:])
		}
		if msg == "" && err != nil {
			msg = err.Error()
		}
		return fmt.Errorf("Windows no lo borró: %s", msg)
	}
	return nil
}

// Demo son datos de ejemplo para el modo demostración.
func Demo() []Pkg {
	return []Pkg{
		{Name: "oem41.inf", Original: "u0412345.inf", Provider: "Advanced Micro Devices, Inc.", Class: "Display", Date: "2025-06-10", Version: "32.0.21001.9024", SizeMB: 1630, Newer: "32.0.22029.1019"},
		{Name: "oem18.inf", Original: "rt640x64.inf", Provider: "Realtek", Class: "Net", Date: "2024-11-03", Version: "10.68.815.2024", SizeMB: 4, Newer: "10.72.1112.2025"},
	}
}
