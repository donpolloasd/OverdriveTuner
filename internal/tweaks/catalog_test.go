package tweaks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCatalogSanity(t *testing.T) {
	all := All()
	if len(all) < 50 {
		t.Fatalf("se esperaban >=50 tweaks, hay %d", len(all))
	}
	ids := map[string]bool{}
	cats := map[string]bool{}
	for _, c := range Categories() {
		cats[c.ID] = true
	}
	for _, tw := range all {
		if ids[tw.ID] {
			t.Errorf("id duplicado %s", tw.ID)
		}
		ids[tw.ID] = true
		if !cats[tw.Cat] {
			t.Errorf("%s: categoría desconocida %s", tw.ID, tw.Cat)
		}
		if len(tw.Ops) == 0 {
			t.Errorf("%s: sin operaciones", tw.ID)
		}
		for _, op := range tw.Ops {
			if op.Kind == OpReg && (op.Key == "" || op.Type == "" || op.Def == "") {
				t.Errorf("%s: op de registro incompleta %+v", tw.ID, op)
			}
		}
	}
	t.Logf("tweaks: %d", len(all))
}

// Los ajustes de la versión 1.0 no pueden cambiar de lugar ni de significado:
// las copias de seguridad guardadas por los usuarios se indexan por posición.
func TestBaselineV10Preserved(t *testing.T) {
	raw, err := os.ReadFile("testdata/baseline_v1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	var base map[string][]string
	if err := json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	for id, ops := range base {
		tw, ok := ByID(id)
		if !ok {
			t.Errorf("se eliminó el ajuste %s de la versión 1.0", id)
			continue
		}
		if len(tw.Ops) < len(ops) {
			t.Errorf("%s: ahora tiene menos operaciones (%d < %d)", id, len(tw.Ops), len(ops))
			continue
		}
		for i, want := range ops {
			o := tw.Ops[i]
			got := fmt.Sprintf("%s|%s|%s|%s|%s|%d", o.Kind, o.Key, o.Name, o.Type, o.Svc, o.Start)
			if got != want {
				t.Errorf("%s op %d cambió:\n  1.0: %s\n  ahora: %s", id, i, want, got)
			}
		}
	}
}

// Dos ajustes no pueden ser dueños del mismo valor de registro ni del mismo
// servicio: al revertir en distinto orden dejarían un estado incoherente.
func TestNoOverlappingOwnership(t *testing.T) {
	owner := map[string]string{}
	for _, tw := range All() {
		for _, op := range tw.Ops {
			var k string
			switch op.Kind {
			case OpReg:
				k = strings.ToUpper(op.Key) + "|" + strings.ToLower(op.Name)
			case OpSvc:
				k = "SVC|" + strings.ToLower(op.Svc)
			default:
				continue
			}
			if prev, dup := owner[k]; dup && prev != tw.ID {
				// los ajustes de un mismo grupo exclusivo comparten valor: solo uno puede estar activo
				if pt, _ := ByID(prev); pt.Group == "" || pt.Group != tw.Group {
					t.Errorf("%q lo gestionan %s y %s", k, prev, tw.ID)
				}
			}
			owner[k] = tw.ID
		}
	}
}

// Ningún ajuste puede tocar lo que exigen los anti-cheat (FACEIT, Vanguard…)
// sin declararlo con AC (que obliga a Risk 2 y doble confirmación).
func TestAntiCheatProtections(t *testing.T) {
	sens := []string{
		"deviceguard", "hypervisorenforcedcodeintegrity", "hypervisorlaunchtype", "securebootstate",
		`control\secureboot`, "\\tpm", "tbs", "dmaguard", "kerneldmaprotection", "iommu",
		"testsigning", "nointegritychecks", "loadoptions", "wuauserv", "usosvc", "waasmedicsvc",
		"bits", "windefend", "sense", "wdfilter", "wdboot", "securityhealthservice", "vulnerabledriverblocklistenable",
		"hvci", "bcdedit_nx", "nx",
	}
	for _, tw := range All() {
		var blob string
		for _, op := range tw.Ops {
			blob += strings.ToLower(op.Key+"|"+op.Name+"|"+op.Svc+"|"+op.Exe+" "+strings.Join(op.Args, " ")+"|"+op.Apply+"|"+op.Revert) + "\n"
		}
		hit := ""
		for _, k := range sens {
			if k == "nx" || k == "bcdedit_nx" || k == "bits" || k == "sense" || k == "tbs" || k == `\tpm` {
				// palabras cortas: solo como valor/servicio exacto para evitar falsos positivos
				for _, op := range tw.Ops {
					if strings.EqualFold(op.Svc, k) || strings.EqualFold(op.Name, k) {
						hit = k
					}
				}
				if k == `\tpm` && strings.Contains(blob, `\tpm`) {
					hit = k
				}
				continue
			}
			if strings.Contains(blob, k) {
				hit = k
			}
		}
		if hit != "" && tw.AC == "" {
			t.Errorf("%s toca «%s» (anti-cheat) y no declara AC", tw.ID, hit)
		}
		if tw.AC != "" && (tw.Risk < 2 || tw.Rec || tw.Comp) {
			t.Errorf("%s: un ajuste AC debe ser Risk 2 y fuera de los perfiles", tw.ID)
		}
	}
	if tw, _ := ByID("adv.vbs"); tw.AC == "" {
		t.Error("adv.vbs debe llevar AC")
	}
	if tw, _ := ByID("adv.hypervisor"); tw.AC == "" {
		t.Error("adv.hypervisor debe llevar AC")
	}
}
