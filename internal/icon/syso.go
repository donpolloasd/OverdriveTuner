package icon

import (
	"bytes"
	"encoding/binary"
	"sort"
)

type resource struct {
	typ  uint16
	id   uint16
	data []byte
}

// Manifest devuelve el manifiesto de aplicación; admin pide ejecutar como administrador.
func Manifest(admin bool, name string) []byte {
	level := "asInvoker"
	if admin {
		level = "requireAdministrator"
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="OverdriveTuner.` + name + `" version="1.0.0.0" processorArchitecture="*"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security><requestedPrivileges><requestedExecutionLevel level="` + level + `" uiAccess="false"/></requestedPrivileges></security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
      <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings><longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware></windowsSettings>
  </application>
</assembly>`)
}

// Syso genera un objeto COFF (amd64) con el icono y el manifiesto como
// recursos, listo para que el enlazador de Go lo incluya en el .exe.
func Syso(admin bool, name string) []byte {
	var res []resource
	// Imágenes del icono (RT_ICON = 3) y su grupo (RT_GROUP_ICON = 14).
	var grp bytes.Buffer
	_ = binary.Write(&grp, binary.LittleEndian, [3]uint16{0, 1, uint16(len(Sizes))})
	for i, s := range Sizes {
		d := imageData(s)
		res = append(res, resource{3, uint16(i + 1), d})
		grp.Write([]byte{dim(s), dim(s), 0, 0})
		_ = binary.Write(&grp, binary.LittleEndian, uint16(1))
		_ = binary.Write(&grp, binary.LittleEndian, uint16(32))
		_ = binary.Write(&grp, binary.LittleEndian, uint32(len(d)))
		_ = binary.Write(&grp, binary.LittleEndian, uint16(i+1))
	}
	res = append(res, resource{14, 1, grp.Bytes()})
	res = append(res, resource{24, 1, Manifest(admin, name)})
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].typ != res[j].typ {
			return res[i].typ < res[j].typ
		}
		return res[i].id < res[j].id
	})

	// Agrupar por tipo.
	var types []uint16
	byType := map[uint16][]resource{}
	for _, r := range res {
		if _, ok := byType[r.typ]; !ok {
			types = append(types, r.typ)
		}
		byType[r.typ] = append(byType[r.typ], r)
	}

	// Calcular posiciones: raíz, un subdirectorio por tipo, un subdirectorio
	// de idioma por recurso, entradas de datos y luego los datos.
	dirSize := func(n int) int { return 16 + 8*n }
	off := dirSize(len(types))
	typeOff := map[uint16]int{}
	for _, t := range types {
		typeOff[t] = off
		off += dirSize(len(byType[t]))
	}
	langOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			langOff[[2]uint16{r.typ, r.id}] = off
			off += dirSize(1)
		}
	}
	entryOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			entryOff[[2]uint16{r.typ, r.id}] = off
			off += 16
		}
	}
	dataOff := map[[2]uint16]int{}
	for _, t := range types {
		for _, r := range byType[t] {
			off = (off + 3) &^ 3
			dataOff[[2]uint16{r.typ, r.id}] = off
			off += len(r.data)
		}
	}
	total := (off + 3) &^ 3

	sec := make([]byte, total)
	put16 := func(o int, v uint16) { binary.LittleEndian.PutUint16(sec[o:], v) }
	put32 := func(o int, v uint32) { binary.LittleEndian.PutUint32(sec[o:], v) }
	dirHeader := func(o, ids int) { put16(o+12, 0); put16(o+14, uint16(ids)) }

	dirHeader(0, len(types))
	for i, t := range types {
		e := 16 + 8*i
		put32(e, uint32(t))
		put32(e+4, 0x80000000|uint32(typeOff[t]))
		td := typeOff[t]
		dirHeader(td, len(byType[t]))
		for j, r := range byType[t] {
			k := [2]uint16{r.typ, r.id}
			ee := td + 16 + 8*j
			put32(ee, uint32(r.id))
			put32(ee+4, 0x80000000|uint32(langOff[k]))
			ld := langOff[k]
			dirHeader(ld, 1)
			put32(ld+16, 0x0409) // inglés (EE. UU.)
			put32(ld+20, uint32(entryOff[k]))
		}
	}
	var relocs []int
	for _, t := range types {
		for _, r := range byType[t] {
			k := [2]uint16{r.typ, r.id}
			eo := entryOff[k]
			put32(eo, uint32(dataOff[k])) // RVA relativo a la sección; el enlazador suma la base
			put32(eo+4, uint32(len(r.data)))
			put32(eo+8, 0)
			copy(sec[dataOff[k]:], r.data)
			relocs = append(relocs, eo)
		}
	}

	var out bytes.Buffer
	w := func(v any) { _ = binary.Write(&out, binary.LittleEndian, v) }
	const hdrLen, secLen = 20, 40
	rawPtr := hdrLen + secLen
	relocPtr := rawPtr + len(sec)
	symPtr := relocPtr + 10*len(relocs)
	w(uint16(0x8664)) // IMAGE_FILE_MACHINE_AMD64
	w(uint16(1))
	w(uint32(0))
	w(uint32(symPtr))
	w(uint32(1))
	w(uint16(0))
	w(uint16(0))
	out.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	w(uint32(0)) // VirtualSize
	w(uint32(0)) // VirtualAddress
	w(uint32(len(sec)))
	w(uint32(rawPtr))
	w(uint32(relocPtr))
	w(uint32(0))
	w(uint16(len(relocs)))
	w(uint16(0))
	w(uint32(0xC0300040))
	out.Write(sec)
	for _, r := range relocs {
		w(uint32(r))
		w(uint32(0)) // símbolo 0 (.rsrc)
		w(uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
	}
	out.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	w(uint32(0))
	w(uint16(1))
	w(uint16(0))
	w(uint8(3)) // IMAGE_SYM_CLASS_STATIC
	w(uint8(0))
	w(uint32(4)) // tabla de cadenas vacía
	return out.Bytes()
}
