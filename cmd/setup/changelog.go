package main

import (
	"strconv"
	"strings"
)

// Change es un cambio explicado en lenguaje simple.
type Change struct {
	T string `json:"t"` // qué cambió
	X string `json:"x"` // por qué importa y cómo se usa
}

// Release agrupa los cambios de una versión.
type Release struct {
	Version string   `json:"version"`
	Title   string   `json:"title"`
	Changes []Change `json:"changes"`
}

// releases va de la más nueva a la más vieja.
var releases = []Release{
	{"2.7.5", "Actualizaciones automáticas", []Change{
		{"Se actualiza solo desde GitHub", "Si lo dejás activado, cuando sale una versión nueva en GitHub el programa la descarga, comprueba su SHA-256 y la instala sin que hagas nada (también desde el overlay que arranca con Windows). Solo consulta GitHub, sin enviar datos tuyos."},
		{"Vos decidís en el instalador", "El Setup tiene una casilla para activarlas. Si la desactivás, el programa se queda en esa versión y solo cambia si aplicás el Parche. Se puede cambiar después en Configuración, donde también hay «Buscar actualizaciones»."},
		{"PresentMon eliminado", "Ya no se incluye PresentMon: los FPS salen de RTSS y la temperatura del procesador de Core Temp. El programa pesa menos."},
	}},
	{"2.7.4", "Arranque más rápido", []Change{
		{"La ventana aparece completa al abrir", "El programa guarda el último escaneo de tu equipo y lo muestra al instante al abrirse; el escaneo se repite en segundo plano. Antes había que esperar varios segundos."},
		{"Instalación y parche más ágiles", "Los accesos directos se crean en una sola llamada a PowerShell, el parche ya no recrea los que existen y no se espera al cerrar un programa que no estaba abierto."},
	}},
	{"2.7.3", "Más seguro", []Change{
		{"El desinstalador solo borra lo suyo", "Antes borraba la carpeta de instalación completa. Si habías instalado dentro de una carpeta que ya tenía otras cosas, también se borraban. Ahora quita solo los archivos de OverdriveTuner y la carpeta únicamente si queda vacía."},
		{"Nombres de programa más estrictos", "Los nombres de .exe (juegos, lista del Modo juego, configuración importada) ya no pueden llevar comillas simples, $ ni acentos graves, que PowerShell interpreta dentro de los comandos."},
	}},
	{"2.7.2", "FPS estables", []Change{
		{"Se corrigió el contador de FPS que se rompía", "Al abrir o reabrir un juego, el overlay podía detener su lector de RTSS por un error interno («close of closed channel») y quedarse sin FPS. Ya no pasa."},
	}},
	{"2.7.1", "Parche sin pantalla de instalación", []Change{
		{"Se corrigió la ventana del Parche", "Al abrir el Parche ya no aparece un instante la pantalla del instalador (con las opciones de RTSS y Core Temp). Muestra solo la del parche."},
	}},
	{"2.7.0", "Core Temp ya configurado", []Change{
		{"Sin íconos por núcleo", "Al instalar Core Temp (desde el Setup o «Instalar programas») se deja una configuración lista: sin un ícono por núcleo en la bandeja y con la ventana minimizada. Si ya tenés Core Temp o su CoreTemp.ini, no se toca nada."},
	}},
	{"2.6.9", "Detección de juegos", []Change{
		{"Reconoce juegos en ventana sin conocer su nombre", "Si abrís un juego que no está en la lista (por ejemplo Minecraft con Java, que es javaw.exe), el overlay lo toma como juego cuando la ventana activa está usando fuerte la placa de video. Ya no hace falta escribir el nombre del .exe."},
	}},
	{"2.6.8", "Ventana maximizada", []Change{
		{"Se abre a pantalla completa", "OverdriveTuner, el Setup y el Parche se abren maximizados, ocupando toda la pantalla (con su barra de título para minimizar o cerrar)."},
	}},
	{"2.6.7", "Parche más simple", []Change{
		{"Menos opciones al parchear", "El parche ya no ofrece instalar RTSS ni Core Temp: solo actualiza OverdriveTuner y te deja elegir si se inicia solo al prender la PC e iniciar sesión."},
	}},
	{"2.6.6", "Core Temp incluido y FPS solo con RTSS", []Change{
		{"Los FPS salen solo de RTSS", "Se quitó la opción de PresentMon: el overlay lee los FPS de RTSS y la temperatura del procesador de Core Temp. Se quitó también el selector «Fuente de FPS»."},
		{"Web oficial si winget falla", "Si winget no está o no puede instalar un programa, en «Instalar programas» aparece el botón «Abrir web oficial» para descargarlo vos desde el sitio del fabricante."},
		{"Cómo funciona en todas las secciones", "Se agregó el tutorial de Modo juego y el de Instalar programas, y se corrigieron los de Overlay, Configuración y las categorías de ajustes."},
		{"Core Temp con la app", "El instalador y el parche pueden instalar Core Temp (casilla marcada por defecto) y el overlay lo abre solo en segundo plano, como hace con RTSS, para mostrar la temperatura del procesador sin que tengas que abrir nada."},
	}},
	{"2.6.5", "Menú del ícono del overlay", []Change{
		{"«Cerrar solo el overlay»", "El ícono del overlay que arranca con Windows ahora dice «Cerrar solo el overlay» en su menú, para que quede claro que no cierra la aplicación principal. Solo cierra el overlay."},
	}},
	{"2.6.4", "Desinstalador nuevo", []Change{
		{"Ventana de desinstalación", "Reemplaza los cuadros de Windows por una ventana limpia, igual a la del instalador. Muestra qué se quita (programa, accesos directos, entrada en Aplicaciones instaladas, inicio con Windows) y qué se conserva (tus juegos, tus ajustes de Windows y tus datos)."},
		{"Tus datos, a elección", "Una casilla permite borrar también tus datos (ajustes, copias de seguridad, pruebas e historial). Por defecto se conservan. Solo se borra la carpeta de datos de OverdriveTuner, nada más."},
		{"Aviso de ajustes aplicados", "Si tenés ajustes aplicados, el desinstalador te avisa cuántos son, que siguen activos en Windows y cómo deshacerlos antes de desinstalar."},
		{"Cierra todo antes de borrar", "Cierra OverdriveTuner, el overlay y PresentMon para que no queden archivos en uso, y quita la tarea de inicio con Windows."},
	}},
	{"2.6.3", "Inicio con Windows más claro", []Change{
		{"Ícono en la bandeja", "Cuando el overlay arranca solo con Windows, ahora muestra un ícono junto al reloj y un aviso, porque sin juego abierto no tiene ventana y no se notaba que estaba activo. Clic en el ícono abre OverdriveTuner."},
		{"Estado y prueba en Configuración", "En «Inicio con Windows» ahora se ve si la tarea está registrada, cuándo se ejecutó por última vez y cómo terminó, y si el overlay está activo. El botón «Probar ahora» la ejecuta sin reiniciar la PC."},
		{"Carpeta de trabajo de la tarea", "La tarea ahora arranca el programa desde su propia carpeta, y el overlay deja una línea en el registro (overdrivetuner.log) cuando se inicia."},
	}},
	{"2.6.2", "Temperatura del procesador con Core Temp", []Change{
		{"Temperatura del CPU sin Afterburner", "El overlay ahora lee la temperatura del procesador de Core Temp (gratis y sin límite de horas), además de MSI Afterburner. Alcanza con tener Core Temp abierto; está en «Instalar programas». Solo lee: no entra al juego. Alimenta también la alerta de sobrecalentamiento."},
	}},
	{"2.6.0", "Modo juego, alertas y arranque con Windows", []Change{
		{"El overlay arranca con Windows", "Al encender la PC el overlay se inicia solo, en segundo plano y sin ventana, y aparece cuando abrís un juego. Usa una tarea programada con permisos de administrador, así que Windows no pide confirmación cada vez. Se desactiva en Configuración, «Inicio con Windows»."},
		{"Modo juego", "Sección nueva donde elegís los programas que se cierran sí o sí cuando se abre un juego. La lista se guarda y se puede editar. Nunca se cierra el juego ni procesos de Windows, y los launchers o anti-cheat piden confirmación."},
		{"Alerta de sobrecalentamiento", "Si el procesador o la gráfica pasan el límite que configures (por defecto CPU 90 °C y GPU 85 °C), el valor se pone en rojo y el overlay muestra un aviso de texto, sin sonido. La de la gráfica funciona siempre; la del procesador necesita Core Temp o Afterburner abiertos."},
		{"Más juegos detectados", "El overlay reconoce también los juegos que EA, GOG y Ubisoft guardan en el registro de Windows, aunque estén en otro disco o carpeta."},
		{"Botón DRIVERS", "Abajo del menú: abre la página oficial de drivers de NVIDIA, AMD o Intel en tu navegador y marca cuál es el de tu PC."},
		{"Monitor en el Diagnóstico", "Ya no se supone que el monitor es de 240 Hz: se usa la frecuencia máxima que Windows ofrece a tu monitor. Si querés otro valor, lo guardás a mano y podés volver al detectado."},
		{"Más programas", "Medal, Lossless Scaling, FurMark, Ryzen Master, WhatsApp, TeamSpeak, Google Drive, OneDrive y AnyDesk. Los que no se instalan con winget abren su página oficial. El software de AMD, NVIDIA o Intel aparece solo si tu PC tiene esa marca."},
		{"Instalador y parche", "Casilla para iniciar el overlay con Windows (activada por defecto)."},
	}},
	{"2.5.0", "Liberar RAM, VRAM y más programas", []Change{
		{"Liberar memoria RAM", "En «Limpieza» hay un botón nuevo. Como «Acelerar» de Microsoft PC Manager, le pide a Windows que devuelva la memoria que los programas abiertos no usan y muestra cuánta había libre antes y después. Es temporal y no toca el juego que tengas abierto."},
		{"Detección de la VRAM", "Si Windows no informa la memoria de la placa de video por el método de siempre, ahora se la pide a DirectX. Así aparece cuánta VRAM tiene tu gráfica en Inicio."},
		{"Instalar programas", "Se quitaron los avisos de anti-cheat y se sumaron más de veinte programas: CrystalDiskInfo y CrystalDiskMark, OCCT, LatencyMon, Cinebench, RTSS solo, Logitech G HUB, Razer Synapse, SteelSeries GG, ShareX, Streamlabs, Audacity, HandBrake, Bitwarden, Rufus, Ventoy, LibreOffice, Java, Git, Python, Node.js y otros."},
		{"Instalador y parche", "El instalador avisa que el overlay puede no medir FPS con juegos de anti-cheat de kernel si no tenés RTSS. Esta ventana del parche explica qué cambia en cada versión."},
	}},
	{"2.4.1", "Fuente de FPS", []Change{
		{"Fuente de FPS", "En la pestaña Overlay podés elegir de dónde salen los FPS: Automática, Solo PresentMon (100 % externo, sin Afterburner) o Solo RTSS. Se aplica la próxima vez que abras un juego."},
		{"Guía «Cómo se miden los FPS»", "Un tutorial animado que explica quién mide y quién muestra, por qué ayuda tener Afterburner o RTSS y cuándo puede no funcionar."},
	}},
	{"2.3.0", "Overlay automático", []Change{
		{"Detección automática del juego", "El overlay reconoce solo qué juego abriste: los conocidos, cualquier juego instalado desde Steam, Epic, EA, GOG, Ubisoft o Xbox (en cualquier disco), los de carpetas que agregues vos y los programas a pantalla completa. Aparece al abrir el juego y se esconde al cerrarlo."},
		{"Se abre solo", "El overlay arranca cuando abrís OverdriveTuner y queda en segundo plano, también en la bandeja."},
		{"Más FPS por segundo", "El contador se actualiza hasta 20 veces por segundo (por defecto 10) y se puede elegir la velocidad."},
		{"Temperaturas desde Afterburner", "Si Windows no informa la temperatura del procesador o de la gráfica y MSI Afterburner está abierto, el overlay las lee de ahí. Solo lee; no toca el juego."},
		{"Más estable con RTSS", "La lectura de RTSS se reforzó para que un error nunca cierre el overlay, y el diagnóstico anota qué fuente se usó."},
	}},
	{"2.2.0", "Rediseño de la interfaz", []Change{
		{"Nueva interfaz", "Sin emojis, en blanco y negro, con listas claras, menú agrupado, diálogos propios y la doble confirmación de anti-cheat dentro de la aplicación."},
		{"Programas nuevos", "FACEIT y su anti-cheat, Spotify, WizTree, Malwarebytes, Visual Studio Code e Intel Driver & Support Assistant."},
	}},
}

// vparts convierte «2.4.1» en [2 4 1].
func vparts(v string) []int {
	var out []int
	for _, p := range strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// newer dice si a es una versión más nueva que b.
func newer(a, b string) bool {
	x, y := vparts(a), vparts(b)
	if x == nil || y == nil {
		return false
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			return p > q
		}
	}
	return false
}

// releasesSince devuelve las versiones posteriores a la instalada (todas si no se sabe cuál es).
func releasesSince(installed string) []Release {
	if vparts(installed) == nil {
		return releases
	}
	var out []Release
	for _, r := range releases {
		if newer(r.Version, installed) {
			out = append(out, r)
		}
	}
	return out
}
