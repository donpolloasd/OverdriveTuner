package tools

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGamePing(t *testing.T) {
	var calls int32
	dial := func(ctx context.Context, addr string) error {
		atomic.AddInt32(&calls, 1)
		switch {
		case strings.HasPrefix(addr, "caido."):
			return errors.New("sin respuesta")
		case strings.HasPrefix(addr, "perdida."):
			if atomic.LoadInt32(&calls)%2 == 0 {
				return errors.New("timeout")
			}
		case strings.HasPrefix(addr, "lento."):
			time.Sleep(20 * time.Millisecond)
		}
		return nil
	}
	targets := []GameTarget{
		{"games", "Lento", "lento.example"},
		{"games", "Rápido", "rapido.example"},
		{"regions", "Caído", "caido.example"},
		{"regions", "Perdida", "perdida.example"},
	}
	res := GamePingTest(dial, targets, 6, time.Millisecond, time.Second)
	if len(res) != 4 {
		t.Fatalf("resultados: %d", len(res))
	}
	if res[0].Name != "Rápido" || res[1].Name != "Lento" {
		t.Errorf("dentro del grupo se ordena por ping: %+v", res[:2])
	}
	if res[1].Avg < 15 || res[1].Status != "ok" {
		t.Errorf("lento: %+v", res[1])
	}
	var caido, perdida *GamePing
	for i := range res {
		switch res[i].Name {
		case "Caído":
			caido = &res[i]
		case "Perdida":
			perdida = &res[i]
		}
	}
	if caido.Status != "error" || caido.LossPc != 100 {
		t.Errorf("caído: %+v", caido)
	}
	if perdida.LossPc == 0 || perdida.Status == "ok" {
		t.Errorf("con pérdidas no debe verse como ok: %+v", perdida)
	}
	if res[len(res)-1].Name != "Caído" {
		t.Error("los que no responden van al final de su grupo")
	}
	if len(GameTargets) < 10 {
		t.Error("faltan destinos")
	}
}
