// Package icon dibuja el icono de la aplicación (sin dependencias externas)
// y lo empaqueta como .ico y como recurso de Windows.
package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Geometría del logo (coordenadas de 0 a 1): un velocímetro en forma de «O»
// abierta abajo, con una aguja que apunta a la zona de «overdrive».
const (
	cx, cy    = 0.5, 0.53
	ringR     = 0.295
	ringHalfW = 0.042
	needleLen = 0.235
	needleW   = 0.042
	hubR      = 0.07
	arcStart  = 135.0 // grados (0 = derecha, sentido horario): inicio del arco
	arcSweep  = 270.0
)

var needleDir = [2]float64{math.Cos(-52 * math.Pi / 180), math.Sin(-52 * math.Pi / 180)}

func inRound(x, y, r float64) bool {
	// Cuadrado redondeado que ocupa todo [0,1].
	px := math.Min(math.Max(x, r), 1-r)
	py := math.Min(math.Max(y, r), 1-r)
	return (x-px)*(x-px)+(y-py)*(y-py) <= r*r
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func segDist(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	t := ((px-ax)*vx + (py-ay)*vy) / (vx*vx + vy*vy)
	t = math.Max(0, math.Min(1, t))
	dx, dy := px-(ax+t*vx), py-(ay+t*vy)
	return math.Sqrt(dx*dx + dy*dy)
}

// pixel devuelve el color (gris, 0 si es fondo) de un punto del logo.
func pixel(x, y float64) (gray float64, ink bool) {
	dx, dy := x-cx, y-cy
	r := math.Hypot(dx, dy)
	// Arco: del gris oscuro (lento) al blanco (overdrive).
	ang := math.Atan2(dy, dx) * 180 / math.Pi
	if ang < 0 {
		ang += 360
	}
	t := ang - arcStart
	if t < 0 {
		t += 360
	}
	inArc := t <= arcSweep && math.Abs(r-ringR) <= ringHalfW
	if !inArc { // puntas redondeadas
		for _, e := range []float64{arcStart, arcStart + arcSweep} {
			ex, ey := cx+ringR*math.Cos(e*math.Pi/180), cy+ringR*math.Sin(e*math.Pi/180)
			if math.Hypot(x-ex, y-ey) <= ringHalfW {
				inArc = true
				t = math.Max(0, math.Min(arcSweep, t))
				if e == arcStart {
					t = 0
				} else {
					t = arcSweep
				}
			}
		}
	}
	if inArc {
		u := math.Max(0, math.Min(1, t/arcSweep))
		return lerp(70, 255, u*u), true
	}
	// Aguja y centro.
	tipX, tipY := cx+needleDir[0]*needleLen, cy+needleDir[1]*needleLen
	if segDist(x, y, cx, cy, tipX, tipY) <= needleW/2 {
		return 255, true
	}
	if r <= hubR {
		if r <= hubR*0.36 {
			return 0, false // agujero del centro
		}
		return 255, true
	}
	return 0, false
}

// Render dibuja el icono a size×size (suavizado con supersampling 4×4).
func Render(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var g, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/ss) / float64(size)
					if !inRound(x, y, 0.22) {
						continue
					}
					c := 10.0 // fondo casi negro
					if gray, ink := pixel(x, y); ink {
						c = gray
					} else if !inRound(x, y, 0.22) || !inRoundInset(x, y) {
						c = 38 // borde sutil
					}
					g, a = g+c, a+1
				}
			}
			if a > 0 {
				v := uint8(g / a)
				img.SetNRGBA(px, py, color.NRGBA{v, v, v, uint8(a * 255 / (ss * ss))})
			}
		}
	}
	return img
}

// inRoundInset dibuja un borde fino de 1,5 % dentro del cuadrado redondeado.
func inRoundInset(x, y float64) bool {
	const m = 0.018
	return x >= m && x <= 1-m && y >= m && y <= 1-m && inRound((x-m)/(1-2*m), (y-m)/(1-2*m), 0.22)
}

// PNG devuelve el logo como PNG de size×size.
func PNG(size int) []byte { return pngBytes(size) }

// Sizes que incluye el icono.
var Sizes = []int{16, 32, 48, 256}

func pngBytes(size int) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, Render(size))
	return buf.Bytes()
}

// dib codifica la imagen como DIB de 32 bits (sin cabecera BMP) + máscara AND.
func dib(size int) []byte {
	img := Render(size)
	var buf bytes.Buffer
	hdr := []any{uint32(40), int32(size), int32(size * 2), uint16(1), uint16(32), uint32(0), uint32(0), int32(0), int32(0), uint32(0), uint32(0)}
	for _, v := range hdr {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	rowMask := ((size + 31) / 32) * 4
	buf.Write(make([]byte, rowMask*size))
	return buf.Bytes()
}

// imageData devuelve los bytes de una imagen del icono: PNG para 256 px, DIB para el resto.
func imageData(size int) []byte {
	if size >= 256 {
		return pngBytes(size)
	}
	return dib(size)
}

func dim(size int) byte {
	if size >= 256 {
		return 0
	}
	return byte(size)
}

// ICO devuelve el contenido de un archivo .ico.
func ICO() []byte {
	var buf bytes.Buffer
	n := len(Sizes)
	_ = binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, uint16(n)})
	offset := 6 + 16*n
	datas := make([][]byte, n)
	for i, s := range Sizes {
		datas[i] = imageData(s)
		buf.Write([]byte{dim(s), dim(s), 0, 0})
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(datas[i])))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(datas[i])
	}
	for _, d := range datas {
		buf.Write(d)
	}
	return buf.Bytes()
}
