// Package tweaks contiene el catálogo de optimizaciones y el motor que las
// aplica y revierte de forma segura.
package tweaks

import "time"

// OpKind es el tipo de operación elemental de un tweak.
type OpKind string

const (
	OpReg  OpKind = "reg"  // valor del registro
	OpSvc  OpKind = "svc"  // tipo de inicio de un servicio
	OpPS   OpKind = "ps"   // script de PowerShell con su reversión
	OpCmd  OpKind = "cmd"  // ejecutable con argumentos y su reversión
	OpPlan OpKind = "plan" // plan de energía propio
)

// Del indica que el valor por defecto de Windows es "no existe".
const Del = "<del>"

// Op es una operación elemental.
type Op struct {
	Kind     OpKind
	Optional bool // si falla no cuenta como error (p. ej. claves protegidas por Windows)
	Aux      bool // operación auxiliar: no se usa para saber si el tweak está aplicado

	// OpReg
	Key, Name, Type, Data string
	Def                   string // valor de fábrica; Del = borrar
	RevDelKey             string // al revertir, si el valor no existía, borra esta clave completa

	// OpSvc
	Svc      string
	Start    int // 2 auto, 3 manual, 4 deshabilitado
	DefStart int

	// OpPS
	Apply, Revert string

	// OpCmd
	Exe     string
	Args    []string
	RevArgs []string
	Timeout time.Duration
}

// Tweak es una optimización que el usuario puede activar o desactivar.
type Tweak struct {
	ID     string
	Cat    string
	Name   string
	Desc   string
	Warn   string // advertencia visible
	Risk   int    // 0 seguro, 1 medio, 2 avanzado
	Rec    bool   // incluido en el perfil recomendado
	Comp   bool   // incluido en el perfil competitivo
	Reboot bool   // requiere reiniciar
	Test   bool   // efecto variable según juego/hardware: no entra en perfiles, hay que probarlo
	Cond   string // condición de hardware: ssd, ram16, desktop, laptop, amd, nvidia, win11
	AC     string // si no está vacío: desactiva algo que exigen los anti-cheat (anti-cheat de kernel); pide doble confirmación
	Watch  bool   // se vigila: si Windows deshace el ajuste, la interfaz avisa
	Hidden bool   // retirado del catálogo visible (solo se muestra si ya estaba aplicado)
	Group  string // grupo exclusivo: al aplicar uno se revierten los demás del grupo
	Ops    []Op
}

// Category agrupa tweaks en la interfaz.
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Desc string `json:"desc"`
}

// ---- constructores cortos -------------------------------------------------

// D crea una operación REG_DWORD.
func D(key, name, data, def string) Op {
	return Op{Kind: OpReg, Key: key, Name: name, Type: "REG_DWORD", Data: data, Def: def}
}

// S crea una operación REG_SZ.
func S(key, name, data, def string) Op {
	return Op{Kind: OpReg, Key: key, Name: name, Type: "REG_SZ", Data: data, Def: def}
}

// Opt marca una operación como opcional.
func (o Op) Opt() Op { o.Optional = true; return o }

// DelKey indica una clave completa a borrar al revertir si el valor no existía antes.
func (o Op) DelKey(key string) Op { o.RevDelKey = key; return o }

// Helper marca una operación como auxiliar (p. ej. notificar al sistema un cambio).
func (o Op) Helper() Op { o.Aux = true; return o }

// Svc deshabilita/ajusta un servicio (solo si existe).
func Svc(name string, start, def int) Op {
	return Op{Kind: OpSvc, Svc: name, Start: start, DefStart: def}
}

// PS crea una operación de PowerShell.
func PS(apply, revert string) Op { return Op{Kind: OpPS, Apply: apply, Revert: revert} }

// Cmd crea una operación de comando; Rev define los argumentos de reversión.
func Cmd(exe string, args ...string) Op { return Op{Kind: OpCmd, Exe: exe, Args: args} }

// Rev fija los argumentos para revertir el comando.
func (o Op) Rev(args ...string) Op { o.RevArgs = args; return o }

// Long amplía el tiempo máximo del comando.
func (o Op) Long(d time.Duration) Op { o.Timeout = d; return o }

const svcKey = `HKLM\SYSTEM\CurrentControlSet\Services\`

// Categories devuelve las categorías en orden de visualización.
func Categories() []Category {
	return []Category{
		{"fps", "FPS y juegos", "⚡", "Ajustes directos para ganar fotogramas"},
		{"latency", "Latencia y entrada", "🖱", "Mouse, teclado y temporizadores"},
		{"gpu", "Gráficos", "🎮", "Opciones de la tarjeta gráfica y DWM"},
		{"power", "Energía y CPU", "🔋", "Plan de energía y estados del procesador"},
		{"memory", "Memoria y disco", "💾", "RAM, SSD y sistema de archivos"},
		{"network", "Red y ping", "🌐", "Menor latencia en línea"},
		{"services", "Servicios", "⚙", "Servicios de Windows que no necesitas"},
		{"privacy", "Privacidad", "🔒", "Telemetría y seguimiento"},
		{"debloat", "Debloat de Windows 11", "🧹", "Quita publicidad, widgets y relleno"},
		{"apps", "Apps preinstaladas", "📦", "Desinstala apps que no usas"},
		{"ui", "Interfaz y efectos", "🖥", "Animaciones y efectos visuales"},
		{"comfort", "Estética y comodidad", "🎨", "Detalles de Explorador, teclado y tema (apagados por defecto)"},
		{"updates", "Actualizaciones", "⏸", "Decidí cuándo se actualiza Windows"},
		{"advanced", "Avanzado (riesgo)", "☢", "Solo si sabes lo que haces"},
	}
}
