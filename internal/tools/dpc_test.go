package tools

import "testing"

func TestJudgeDPC(t *testing.T) {
	ok := judgeDPC([]dpcSample{{"_Total", 0.4, 0.2}, {"0,0", 3, 1}, {"0,1", 1, 0.5}, {"0,_Total", 9, 9}}, 10)
	if ok.Status != "ok" || ok.DPCMax != 3 || ok.Core != "0,0" || len(ok.Tips) != 0 {
		t.Fatalf("%+v", ok)
	}
	bad := judgeDPC([]dpcSample{{"_Total", 6, 1}, {"0,3", 40, 2}}, 10)
	if bad.Status != "bad" || bad.Core != "0,3" || len(bad.Tips) == 0 {
		t.Fatalf("%+v", bad)
	}
}
