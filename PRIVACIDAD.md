# Privacidad de OverdriveTuner

OverdriveTuner no tiene cuentas, publicidad ni telemetría. No envía datos de tu equipo, de tus juegos ni de cómo usás el programa a nadie, tampoco a su autor.

Todo lo que el programa sabe de tu PC (hardware, ajustes aplicados, copias de seguridad, benchmarks, historial) se guarda solo en tu equipo, en `C:\ProgramData\FPSForge`.

## Conexiones a internet

El programa solo se conecta a internet en estos casos:

| Para qué | A dónde | Cuándo | Cómo evitarlo |
|---|---|---|---|
| Buscar actualizaciones | `api.github.com` y `github.com` (Releases de este proyecto) | 25 s después de abrir (o 90 s después de que arranque el overlay con Windows) y cada 6 horas, **solo si activaste las actualizaciones automáticas** (casilla del instalador o Configuración → Actualizaciones), o cuando apretás «Buscar actualizaciones» | Dejá desactivadas las actualizaciones automáticas |
| Instalar programas | Repositorio oficial de winget (Microsoft) y las webs de cada programa | Solo cuando elegís instalar algo en «Instalar programas», o RTSS/Core Temp en el instalador | No los instales desde OverdriveTuner |
| Pruebas de red, ping y DNS | Cloudflare, Google, Quad9, OpenDNS y los servidores de juegos y regiones de la lista | Solo cuando apretás el botón de esa prueba | No uses esas pruebas |
| Páginas de drivers y webs oficiales | Se abren en tu navegador (NVIDIA, AMD, Intel y otras) | Solo cuando apretás ese botón | — |

En ninguna de estas conexiones se envían datos personales ni datos de tu equipo. GitHub y los demás servicios pueden ver tu dirección IP, como con cualquier conexión a internet.

## Cambios en el sistema

OverdriveTuner cambia configuraciones de Windows solo cuando vos lo pedís: cada ajuste se aplica con su interruptor, guarda una copia del valor anterior y se puede revertir. Los ajustes avanzados y los que afectan a los anti-cheat piden confirmación. El desinstalador quita el programa; los ajustes aplicados quedan hasta que los revertís.
