// baseline escribe una huella de las operaciones de cada ajuste, para que los
// parches futuros no cambien el significado de copias de seguridad ya guardadas.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"fpsforge/internal/tweaks"
)

func main() {
	out := map[string][]string{}
	for _, t := range tweaks.All() {
		var ops []string
		for _, o := range t.Ops {
			ops = append(ops, fmt.Sprintf("%s|%s|%s|%s|%s|%d", o.Kind, o.Key, o.Name, o.Type, o.Svc, o.Start))
		}
		out[t.ID] = ops
	}
	b, _ := json.MarshalIndent(out, "", " ")
	if err := os.WriteFile(os.Args[1], b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
