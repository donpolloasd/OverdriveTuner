package system

import "testing"

func TestRoundTrip(t *testing.T) {
	cases := []struct{ typ, data string }{
		{"REG_DWORD", "0"}, {"REG_DWORD", "4294967295"}, {"REG_DWORD", "38"},
		{"REG_QWORD", "18446744073709551615"},
		{"REG_SZ", "Hola ñandú ✓"}, {"REG_SZ", ""}, {"REG_SZ", "VRROptimizeEnable=1;SwapEffectUpgradeEnable=1;"},
		{"REG_EXPAND_SZ", `%SystemRoot%\x`},
		{"REG_MULTI_SZ", `uno\0dos\0tres`},
		{"REG_BINARY", "030000000000000000000000"}, {"REG_BINARY", ""},
	}
	for _, c := range cases {
		k, raw, err := EncodeValue(c.typ, c.data)
		if err != nil {
			t.Fatalf("%v: %v", c, err)
		}
		typ, data := DecodeValue(k, raw)
		if typ != c.typ || data != c.data {
			t.Errorf("%v -> (%s, %q)", c, typ, data)
		}
	}
}

func TestEncodeErrors(t *testing.T) {
	for _, c := range []struct{ typ, data string }{
		{"REG_DWORD", "-1"}, {"REG_DWORD", "4294967296"}, {"REG_DWORD", "abc"}, {"REG_BINARY", "xyz"}, {"REG_NONE", ""},
	} {
		if _, _, err := EncodeValue(c.typ, c.data); err == nil {
			t.Errorf("%v debería fallar", c)
		}
	}
}

func TestHexDword(t *testing.T) {
	_, raw, err := EncodeValue("REG_DWORD", "0x26")
	if err != nil || raw[0] != 0x26 {
		t.Fatalf("%v %v", raw, err)
	}
}
