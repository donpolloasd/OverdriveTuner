// Package diskmap analiza qué ocupa lugar en un disco, al estilo de WizTree: recorre las carpetas en
// paralelo y arma el árbol de tamaños, los archivos más grandes, el espacio por tipo de archivo y los
// grupos de archivos repetidos. Solo lee; borrar (siempre a la Papelera) está en recycle.go.
package diskmap

import (
	"container/heap"
	"context"
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Node es una carpeta (con sus subcarpetas) o, en las vistas, un archivo suelto.
type Node struct {
	Name  string
	Size  int64
	Files int
	Kids  []*Node // solo carpetas
}

// File es un archivo con su ruta completa.
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// TypeStat es lo que ocupan los archivos de una extensión.
type TypeStat struct {
	Ext   string `json:"ext"`
	Bytes int64  `json:"bytes"`
	Count int    `json:"count"`
}

// Progress cuenta lo recorrido mientras dura el análisis (se lee desde otro hilo).
type Progress struct {
	Files atomic.Int64
	Bytes atomic.Int64
}

// Result es el análisis completo de una carpeta o unidad.
type Result struct {
	RootPath string
	Root     *Node
	Files    int
	Dirs     int
	Bytes    int64
	Skipped  int // carpetas sin permiso de lectura
	Took     time.Duration

	mu      sync.Mutex
	largest fileHeap
	types   map[string]*TypeStat
	bySize  map[int64][]string // archivos de 1 MB o más, para buscar duplicados
}

const (
	topFiles   = 200
	dupMinSize = 1 << 20 // los duplicados chicos no valen la pena (y serían miles)
	workers    = 16
)

// Scan recorre root. Se puede cancelar con ctx; prog (opcional) se actualiza mientras avanza.
func Scan(ctx context.Context, root string, prog *Progress) (*Result, error) {
	root = filepath.Clean(root)
	if fi, err := os.Stat(root); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, &fs.PathError{Op: "scan", Path: root, Err: fs.ErrInvalid}
	}
	if prog == nil {
		prog = &Progress{}
	}
	start := time.Now()
	r := &Result{RootPath: root, Root: &Node{Name: root}, types: map[string]*TypeStat{}, bySize: map[int64][]string{}}
	sem := make(chan struct{}, workers)
	var walk func(dir string, n *Node)
	walk = func(dir string, n *Node) {
		if ctx.Err() != nil {
			return
		}
		es, err := os.ReadDir(dir)
		if err != nil {
			r.mu.Lock()
			r.Skipped++
			r.mu.Unlock()
			return
		}
		var big []File
		types := map[string]*TypeStat{}
		var dupes []File
		var wg sync.WaitGroup
		for _, e := range es {
			t := e.Type()
			if t.IsDir() {
				k := &Node{Name: e.Name()}
				n.Kids = append(n.Kids, k)
				p := filepath.Join(dir, e.Name())
				select {
				case sem <- struct{}{}: // hay un hilo libre: la subcarpeta se recorre en paralelo
					wg.Add(1)
					go func() { defer wg.Done(); defer func() { <-sem }(); walk(p, k) }()
				default:
					walk(p, k)
				}
				continue
			}
			if !t.IsRegular() { // accesos directos, junctions y demás: no se siguen ni se cuentan
				continue
			}
			fi, err := e.Info()
			if err != nil || cloudOnly(fi) {
				continue
			}
			sz := fi.Size()
			n.Size += sz
			n.Files++
			prog.Files.Add(1)
			prog.Bytes.Add(sz)
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == "" {
				ext = "(sin extensión)"
			}
			ts := types[ext]
			if ts == nil {
				ts = &TypeStat{Ext: ext}
				types[ext] = ts
			}
			ts.Bytes += sz
			ts.Count++
			p := filepath.Join(dir, e.Name())
			if sz >= 1<<20 {
				big = append(big, File{p, sz})
			}
			if sz >= dupMinSize {
				dupes = append(dupes, File{p, sz})
			}
		}
		wg.Wait()
		for _, k := range n.Kids {
			n.Size += k.Size
			n.Files += k.Files
		}
		r.mu.Lock()
		r.Dirs += len(n.Kids)
		for _, f := range big {
			r.largest.push(f)
		}
		for ext, ts := range types {
			if cur := r.types[ext]; cur != nil {
				cur.Bytes += ts.Bytes
				cur.Count += ts.Count
			} else {
				r.types[ext] = ts
			}
		}
		for _, f := range dupes {
			r.bySize[f.Size] = append(r.bySize[f.Size], f.Path)
		}
		r.mu.Unlock()
	}
	walk(root, r.Root)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.Files, r.Bytes, r.Took = r.Root.Files, r.Root.Size, time.Since(start)
	return r, nil
}

// fileHeap guarda los archivos más grandes (un montículo con el más chico arriba).
type fileHeap []File

func (h fileHeap) Len() int           { return len(h) }
func (h fileHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h fileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *fileHeap) Push(x any)        { *h = append(*h, x.(File)) }
func (h *fileHeap) Pop() any          { o := *h; f := o[len(o)-1]; *h = o[:len(o)-1]; return f }

func (h *fileHeap) push(f File) {
	if h.Len() < topFiles {
		heap.Push(h, f)
	} else if f.Size > (*h)[0].Size {
		(*h)[0] = f
		heap.Fix(h, 0)
	}
}

// Largest devuelve los archivos más grandes, del mayor al menor.
func (r *Result) Largest(n int) []File {
	r.mu.Lock()
	out := append([]File(nil), r.largest...)
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// Types devuelve el espacio por tipo de archivo, de lo que más ocupa a lo que menos.
func (r *Result) Types(n int) []TypeStat {
	r.mu.Lock()
	out := make([]TypeStat, 0, len(r.types))
	for _, t := range r.types {
		out = append(out, *t)
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// Find busca la carpeta de una ruta dentro del análisis (sin distinguir mayúsculas).
func (r *Result) Find(path string) *Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.find(path)
}

func (r *Result) find(path string) *Node {
	path = filepath.Clean(path)
	if strings.EqualFold(path, r.RootPath) {
		return r.Root
	}
	rel, err := filepath.Rel(r.RootPath, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil
	}
	n := r.Root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		var next *Node
		for _, k := range n.Kids {
			if strings.EqualFold(k.Name, part) {
				next = k
				break
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

// Entry es un elemento de la vista de una carpeta.
type Entry struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Files int    `json:"files"`
	Dir   bool   `json:"dir"`
}

// View es una carpeta con sus subcarpetas más grandes y, aparte, lo que ocupan sus archivos sueltos.
type View struct {
	Path   string  `json:"path"`
	Size   int64   `json:"size"`
	Files  int     `json:"files"`
	Kids   []Entry `json:"kids"`
	Loose  int64   `json:"loose"` // archivos que están directo en esta carpeta
	Others int64   `json:"others"`
}

// ViewOf arma la vista de una carpeta con hasta limit subcarpetas (el resto se suma en Others).
func (r *Result) ViewOf(path string, limit int) (View, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.find(path)
	if n == nil {
		return View{}, false
	}
	v := View{Path: filepath.Clean(path), Size: n.Size, Files: n.Files, Kids: []Entry{}}
	kids := append([]*Node(nil), n.Kids...)
	sort.Slice(kids, func(i, j int) bool { return kids[i].Size > kids[j].Size })
	var sub int64
	for i, k := range kids {
		sub += k.Size
		if i < limit {
			v.Kids = append(v.Kids, Entry{Name: k.Name, Size: k.Size, Files: k.Files, Dir: true})
		} else {
			v.Others += k.Size
		}
	}
	v.Loose = n.Size - sub
	return v, true
}

// DupGroup son archivos con el mismo contenido.
type DupGroup struct {
	Size   int64    `json:"size"`
	Files  []string `json:"files"`
	Wasted int64    `json:"wasted"` // lo que se liberaría dejando una sola copia
}

// Duplicates busca archivos repetidos (1 MB o más) entre los que tienen el mismo tamaño: primero compara
// el comienzo del archivo y después el contenido completo. Se detiene al llegar a budget o a ctx.
func (r *Result) Duplicates(ctx context.Context, budget time.Duration, maxGroups int) ([]DupGroup, bool) {
	deadline := time.Now().Add(budget)
	r.mu.Lock()
	sizes := make([]int64, 0, len(r.bySize))
	for sz, l := range r.bySize {
		if len(l) > 1 {
			sizes = append(sizes, sz)
		}
	}
	r.mu.Unlock()
	sort.Slice(sizes, func(i, j int) bool { return sizes[i] > sizes[j] }) // los más grandes primero: liberan más
	var out []DupGroup
	complete := true
	for _, sz := range sizes {
		if ctx.Err() != nil || time.Now().After(deadline) || len(out) >= maxGroups {
			complete = false
			break
		}
		r.mu.Lock()
		paths := append([]string(nil), r.bySize[sz]...)
		r.mu.Unlock()
		for _, g := range groupBy(paths, func(p string) string { return hashFile(p, 64<<10) }) {
			for _, full := range groupBy(g, func(p string) string { return hashFile(p, -1) }) {
				out = append(out, DupGroup{Size: sz, Files: full, Wasted: sz * int64(len(full)-1)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Wasted > out[j].Wasted })
	return out, complete
}

// groupBy agrupa por clave y devuelve solo los grupos de dos o más (las claves vacías son errores de lectura).
func groupBy(paths []string, key func(string) string) [][]string {
	m := map[string][]string{}
	var order []string
	for _, p := range paths {
		k := key(p)
		if k == "" {
			continue
		}
		if _, ok := m[k]; !ok {
			order = append(order, k)
		}
		m[k] = append(m[k], p)
	}
	var out [][]string
	for _, k := range order {
		if len(m[k]) > 1 {
			out = append(out, m[k])
		}
	}
	return out
}

// hashFile calcula el SHA-256 de los primeros n bytes (o de todo el archivo si n < 0).
func hashFile(p string, n int64) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	var rd io.Reader = f
	if n >= 0 {
		rd = io.LimitReader(f, n)
	}
	if _, err := io.Copy(h, rd); err != nil {
		return ""
	}
	return string(h.Sum(nil))
}

// Forget descuenta del análisis un archivo o carpeta que se mandó a la Papelera, para que las vistas
// queden al día sin volver a recorrer el disco.
func (r *Result) Forget(path string) {
	path = filepath.Clean(path)
	r.mu.Lock()
	defer r.mu.Unlock()
	var size int64
	var files int
	parentPath := filepath.Dir(path)
	if n := r.find(path); n != nil && n != r.Root {
		size, files = n.Size, n.Files
		if p := r.find(parentPath); p != nil {
			for i, k := range p.Kids {
				if k == n {
					p.Kids = append(p.Kids[:i], p.Kids[i+1:]...)
					break
				}
			}
		}
	} else {
		for _, f := range r.largest {
			if strings.EqualFold(f.Path, path) {
				size, files = f.Size, 1
			}
		}
	}
	inside := func(p string) bool {
		return strings.EqualFold(p, path) || strings.HasPrefix(strings.ToLower(p), strings.ToLower(path)+string(filepath.Separator))
	}
	keep := r.largest[:0]
	for _, f := range r.largest {
		if !inside(f.Path) {
			keep = append(keep, f)
		}
	}
	r.largest = keep
	heap.Init(&r.largest)
	for sz, l := range r.bySize {
		out := l[:0]
		for _, p := range l {
			if !inside(p) {
				out = append(out, p)
			}
		}
		r.bySize[sz] = out
	}
	// Se resta en todas las carpetas que lo contenían, desde la raíz.
	for d := parentPath; ; d = filepath.Dir(d) {
		if n := r.find(d); n != nil {
			n.Size -= size
			n.Files -= files
		}
		if strings.EqualFold(d, r.RootPath) || filepath.Dir(d) == d {
			break
		}
	}
	r.Bytes -= size
	r.Files -= files
}
