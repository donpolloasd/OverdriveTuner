package perf

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDiskMeasuresAndCleansUp(t *testing.T) {
	for _, write := range []bool{false, true} {
		dir := t.TempDir()
		if free := freeBytes(dir); free < 4<<30 {
			t.Skip("poco espacio libre")
		}
		mbs, iops, err := disk(context.Background(), dir, write, 16, 2*time.Second, 200*time.Millisecond, nop)
		if err != nil || mbs <= 0 || iops <= 0 {
			t.Fatal(mbs, iops, err)
		}
		if l, _ := os.ReadDir(dir); len(l) != 0 {
			t.Fatal("el archivo de prueba se borra solo")
		}
		t.Logf("escritura=%v: %.0f MB/s, %.0f IOPS", write, mbs, iops)
	}
}
