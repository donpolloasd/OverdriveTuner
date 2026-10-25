package system

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Tipos de valor del registro de Windows.
const (
	KindSZ       = 1
	KindExpandSZ = 2
	KindBinary   = 3
	KindDWord    = 4
	KindMultiSZ  = 7
	KindQWord    = 11
)

// KindOf devuelve el código numérico de un tipo (REG_DWORD → 4).
func KindOf(typ string) (uint32, error) {
	switch strings.ToUpper(typ) {
	case "REG_SZ":
		return KindSZ, nil
	case "REG_EXPAND_SZ":
		return KindExpandSZ, nil
	case "REG_BINARY":
		return KindBinary, nil
	case "REG_DWORD":
		return KindDWord, nil
	case "REG_MULTI_SZ":
		return KindMultiSZ, nil
	case "REG_QWORD":
		return KindQWord, nil
	}
	return 0, fmt.Errorf("tipo de registro no soportado: %s", typ)
}

// TypeName devuelve el nombre de un tipo a partir de su código.
func TypeName(kind uint32) string {
	switch kind {
	case KindSZ:
		return "REG_SZ"
	case KindExpandSZ:
		return "REG_EXPAND_SZ"
	case KindBinary:
		return "REG_BINARY"
	case KindDWord:
		return "REG_DWORD"
	case KindMultiSZ:
		return "REG_MULTI_SZ"
	case KindQWord:
		return "REG_QWORD"
	}
	return "REG_NONE"
}

func utf16Bytes(s string) []byte {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	b := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	return b
}

func parseUint(s string, bits int) (uint64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		return strconv.ParseUint(s[2:], 16, bits)
	}
	return strconv.ParseUint(s, 10, bits)
}

// EncodeValue convierte (tipo, texto) en los bytes que guarda el registro.
func EncodeValue(typ, data string) (uint32, []byte, error) {
	kind, err := KindOf(typ)
	if err != nil {
		return 0, nil, err
	}
	switch kind {
	case KindDWord:
		n, err := parseUint(data, 32)
		if err != nil {
			return 0, nil, fmt.Errorf("DWORD no válido %q", data)
		}
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(n))
		return kind, b, nil
	case KindQWord:
		n, err := parseUint(data, 64)
		if err != nil {
			return 0, nil, fmt.Errorf("QWORD no válido %q", data)
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, n)
		return kind, b, nil
	case KindSZ, KindExpandSZ:
		return kind, utf16Bytes(data), nil
	case KindMultiSZ:
		var out []byte
		if data != "" {
			for _, p := range strings.Split(data, `\0`) {
				out = append(out, utf16Bytes(p)...)
			}
		}
		out = append(out, 0, 0)
		return kind, out, nil
	case KindBinary:
		h := strings.NewReplacer(" ", "", ",", "").Replace(data)
		b, err := hex.DecodeString(h)
		if err != nil {
			return 0, nil, fmt.Errorf("binario no válido %q", data)
		}
		return kind, b, nil
	}
	return 0, nil, fmt.Errorf("tipo no soportado")
}

func decodeUTF16(b []byte) []string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	var parts []string
	start := 0
	for i, c := range u {
		if c == 0 {
			parts = append(parts, string(utf16.Decode(u[start:i])))
			start = i + 1
		}
	}
	if start < len(u) {
		parts = append(parts, string(utf16.Decode(u[start:])))
	}
	return parts
}

// DecodeValue convierte los bytes del registro en (tipo, texto normalizado).
func DecodeValue(kind uint32, raw []byte) (string, string) {
	typ := TypeName(kind)
	switch kind {
	case KindDWord:
		if len(raw) >= 4 {
			return typ, strconv.FormatUint(uint64(binary.LittleEndian.Uint32(raw)), 10)
		}
	case KindQWord:
		if len(raw) >= 8 {
			return typ, strconv.FormatUint(binary.LittleEndian.Uint64(raw), 10)
		}
	case KindSZ, KindExpandSZ:
		p := decodeUTF16(raw)
		if len(p) == 0 {
			return typ, ""
		}
		return typ, p[0]
	case KindMultiSZ:
		p := decodeUTF16(raw)
		for len(p) > 0 && p[len(p)-1] == "" {
			p = p[:len(p)-1]
		}
		return typ, strings.Join(p, `\0`)
	case KindBinary:
		return typ, hex.EncodeToString(raw)
	}
	return typ, ""
}
