package icon

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Version es la versión que se graba en la ficha del .exe (la fija mkres con la constante del programa).
var Version = "0.0.0"

// productName es el nombre que muestran la barra de tareas, el Administrador de tareas y las propiedades del archivo.
const productName = "OverdriveTuner"

func describe(name string) string {
	if name == "setup" {
		return "Instalador de OverdriveTuner"
	}
	return productName
}

// versionInfo arma el recurso VERSIONINFO (RT_VERSION = 16): sin él, Windows muestra el nombre del archivo
// («OverdriveTuner.exe») en vez del nombre del programa.
func versionInfo(name string) []byte {
	var v [4]uint16
	for i, p := range strings.SplitN(Version, ".", 4) {
		n, _ := strconv.Atoi(p)
		v[i] = uint16(n)
	}
	ms, ls := uint32(v[0])<<16|uint32(v[1]), uint32(v[2])<<16|uint32(v[3])
	fixed := new(bytes.Buffer)
	for _, x := range []uint32{0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x40004, 1, 0, 0, 0} {
		_ = binary.Write(fixed, binary.LittleEndian, x) // VS_FIXEDFILEINFO: firma, versiones, VOS_NT_WINDOWS32, VFT_APP
	}
	desc := describe(name)
	strs := [][2]string{
		{"CompanyName", productName}, {"FileDescription", desc}, {"FileVersion", Version},
		{"InternalName", name}, {"LegalCopyright", "Licencia MIT"}, {"ProductName", productName}, {"ProductVersion", Version},
	}
	var table []byte
	for _, s := range strs {
		table = append(table, block(s[0], u16z(s[1]), 1, uint16(len(utf16.Encode([]rune(s[1])))+1))...)
	}
	sfi := block("StringFileInfo", block("040a04b0", table, 1, 0), 1, 0) // español, Unicode
	vfi := block("VarFileInfo", block("Translation", []byte{0x0a, 0x04, 0xb0, 0x04}, 0, 4), 1, 0)
	return block("VS_VERSION_INFO", append(append(fixed.Bytes(), sfi...), vfi...), 0, uint16(fixed.Len()))
}

// block arma un bloque de VERSIONINFO: longitud, longitud del valor, tipo (1 = texto), clave y contenido,
// alineados a 4 bytes. En VS_VERSION_INFO el valor (VS_FIXEDFILEINFO) va antes que los hijos.
func block(key string, body []byte, typ, valueLen uint16) []byte {
	var b bytes.Buffer
	b.Write([]byte{0, 0, 0, 0, 0, 0})
	b.Write(u16z(key))
	pad4(&b)
	b.Write(body)
	pad4(&b)
	out := b.Bytes()
	binary.LittleEndian.PutUint16(out[0:], uint16(len(out)))
	binary.LittleEndian.PutUint16(out[2:], valueLen)
	binary.LittleEndian.PutUint16(out[4:], typ)
	return out
}

func u16z(s string) []byte {
	var b bytes.Buffer
	for _, r := range utf16.Encode([]rune(s)) {
		_ = binary.Write(&b, binary.LittleEndian, r)
	}
	b.Write([]byte{0, 0})
	return b.Bytes()
}

func pad4(b *bytes.Buffer) {
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
}
