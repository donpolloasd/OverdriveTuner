"use strict";
/* Traducción al inglés. El texto fuente de la interfaz está en español; en inglés se traduce
   al dibujar, con este diccionario. Lo que no esté acá se muestra en español. */
const EXACT = new Map();
const PATTERNS = [];
function T(es, en) {
  if (!es.includes("{")) { EXACT.set(es, en); return; }
  const rx = es.replace(/[.*+?^$()|[\]\\]/g, "\\$&").replace(/\{n\}/g, "(\\d+(?:[.,]\\d+)?)").replace(/\{\}/g, "([\\s\\S]+?)");
  PATTERNS.push([new RegExp("^" + rx + "$"), en]);
}
function fill(en, m) { return en.replace(/\{(\d)\}/g, (_, i) => (m[+i] === undefined ? "" : trText(m[+i]))); }

function trCore(s) {
  if (EXACT.has(s)) return EXACT.get(s);
  for (const [rx, en] of PATTERNS) { const m = s.match(rx); if (m) return fill(en, m); }
  const i = s.indexOf(": ");   // «Etiqueta: texto» con cada mitad traducible por separado
  if (i > 0) { const a = trCore(s.slice(0, i)), b = trCore(s.slice(i + 2)); if (a !== null || b !== null) return (a ?? s.slice(0, i)) + ": " + (b ?? s.slice(i + 2)); }
  return null;
}

/* Traduce un texto completo (conserva espacios de los bordes). Si no hay coincidencia exacta
   prueba por frases y por segmentos separados con « · ». */
function trText(s) {
  if (!s || !/[A-Za-zÁÉÍÓÚáéíóúñ]/.test(s)) return s;
  const lead = s.match(/^\s*/)[0], trail = s.match(/\s*$/)[0];
  let core = s.trim();
  if (!core) return s;
  const pre = core.match(/^([✔✖⚠ℹ✓×•→←↑↓—–-]\s+)(?=\S)/);
  if (pre) { const rest = trText(core.slice(pre[1].length)); return rest === core.slice(pre[1].length) ? s : lead + pre[1] + rest + trail; }
  let r = trCore(core);
  if (r !== null) return lead + r + trail;
  const parts = core.split(/( · |(?<=[.!?:])\s+(?=[A-ZÁÉÍÓÚ¿¡«]))/);
  if (parts.length > 1) {
    let any = false;
    const out = parts.map((p, i) => { if (i % 2) return p; const t = trCore(p.trim()); if (t === null) return p; any = true; return t; });
    if (any) return lead + out.join("") + trail;
  }
  return s;
}

/* ---------------- navegación, inicio y estructura ---------------- */
[
  ["Inicio", "Home"], ["Overlay en juego", "In-game overlay"], ["Diagnóstico", "Diagnostics"], ["Benchmark", "Benchmark"],
  ["Limpieza", "Cleanup"], ["Inicio de Windows", "Windows startup"], ["Historial", "History"], ["Configuración", "Settings"],
  ["Administrador", "Administrator"], ["Sin permisos de administrador", "No administrator rights"],
  ["Cómo funciona", "How it works"], ["Cerrar", "Close"], ["Cancelar", "Cancel"], ["Guardar", "Save"], ["Borrar", "Delete"],
  ["Quitar", "Remove"], ["Ver", "View"], ["Importar", "Import"], ["Importada", "Imported"], ["Exportar", "Export"],
  ["Siguiente", "Next"], ["Anterior", "Previous"], ["Saltar", "Skip"], ["Terminar", "Finish"], ["Ahora no", "Not now"],
  ["Empezar recorrido", "Start tour"], ["¿Primera vez?", "First time?"],
  ["Te muestro en 1 minuto para qué sirve cada botón. Después lo encontrás siempre en «Cómo funciona».", "I'll show you in 1 minute what each button does. You can always find it later under «How it works»."],
  ["Escaneamos tu equipo y solo te ofrecemos lo que le sirve. Todo se puede revertir.", "We scan your PC and only offer what suits it. Everything can be reverted."],
  ["Escaneando tu equipo… ya podés ir mirando las secciones; los perfiles se habilitan cuando termina.", "Scanning your PC… you can already browse the sections; profiles unlock when it finishes."],
  ["Modo de demostración: estás viendo la interfaz fuera de Windows. Los cambios son simulados y no tocan tu sistema.", "Demo mode: you are viewing the interface outside Windows. Changes are simulated and do not touch your system."],
  ["El programa no tiene permisos de administrador, así que no puede aplicar cambios. Ciérralo y ábrelo con clic derecho → «Ejecutar como administrador».", "The program does not have administrator rights, so it cannot apply changes. Close it and open it with right-click → «Run as administrator»."],
  ["Optimiza tu PC en un clic", "Optimize your PC in one click"], ["Tu PC ya está optimizada", "Your PC is already optimized"],
  ["El perfil recomendado aplica solo ajustes seguros para tu hardware. El competitivo suma ajustes de latencia para shooters y esports.", "The recommended profile applies only safe tweaks for your hardware. The competitive one adds latency tweaks for shooters and esports."],
  ["Aplicar perfil recomendado", "Apply recommended profile"], ["Perfil competitivo", "Competitive profile"],
  ["Crear punto de restauración", "Create restore point"], ["Volver a escanear", "Scan again"], ["Restaurar todo", "Restore everything"],
  ["{n} de {n} recomendadas", "{1} of {2} recommended"],
  ["Equipo", "Computer"], ["Procesador", "Processor"], ["Gráficos", "Graphics"], ["Memoria", "Memory"], ["Disco del sistema", "System disk"], ["Disco", "Disk"], ["Sistema", "System"],
  ["{n} núcleos · {n} hilos", "{1} cores · {2} threads"], ["{n} módulos", "{1} modules"], ["compilación {n}", "build {1}"], ["{n} GB de VRAM", "{1} GB of VRAM"],
  ["Punto de restauración", "Restore point"], ["Creando punto de restauración… puede tardar un momento.", "Creating restore point… this may take a moment."],
  ["Creando punto de restauración…", "Creating restore point…"], ["Punto de restauración creado.", "Restore point created."],
  ["Restaurar todo", "Restore everything"], ["Revirtiendo…", "Reverting…"], ["Revertido: ", "Reverted: "], ["No se pudo: ", "Failed: "],
  ["Reiniciá para completar la restauración.", "Restart to complete the restoration."],
  ["No hay nada aplicado por OverdriveTuner.", "Nothing has been applied by OverdriveTuner."],
  ["Se van a revertir los {n} ajustes aplicados por OverdriveTuner y tu sistema quedará como antes de usarlo.\n\n¿Continuar?", "The {1} tweaks applied by OverdriveTuner will be reverted and your system will be as it was before using it.\n\nContinue?"],
  ["Escaneando…", "Scanning…"],
  ["Buscar en esta sección", "Search in this section"], ["Buscar", "Search"],
  ["Aplicar todos los compatibles", "Apply all compatible"], ["Revertir los de esta sección", "Revert the ones in this section"],
  ["{n} de {n} activos", "{1} of {2} active"], ["No hay resultados para esa búsqueda.", "No results for that search."],
  ["Seguro", "Safe"], ["Riesgo medio", "Medium risk"], ["Avanzado", "Advanced"],
  ["Requiere reiniciar", "Requires a restart"], ["Instalar programas", "Install programs"], ["Instalá lo que necesitás con un clic usando winget, el administrador de paquetes de Windows 11. Marcá los programas y tocá «Instalar seleccionados».", "Install what you need in one click using winget, the Windows 11 package manager. Tick the programs and press “Install selected”."], ["Cargando…", "Loading…"], ["Instalado", "Installed"], ["⚠ Anti-cheat", "⚠ Anti-cheat"], ["winget no está disponible en este equipo. Es el administrador de paquetes de Windows 11 que usa esta sección. ", "winget is not available on this PC. It is the Windows 11 package manager this section uses. "], ["Instalar winget", "Install winget"], ["Instalar seleccionados", "Install selected"], ["Actualizar programas instalados", "Update installed programs"], ["Consultando qué ya está instalado…", "Checking what is already installed…"], ["Drivers de video", "Video drivers"], ["Abre la página oficial del fabricante en tu navegador. Los drivers no se instalan desde acá: descargalos y verificalos vos.", "Opens the manufacturer's official page in your browser. Drivers are not installed from here: download and check them yourself."], ["Drivers de NVIDIA", "NVIDIA drivers"], ["Drivers de AMD", "AMD drivers"], ["Drivers de Intel", "Intel drivers"], ["winget listo.", "winget is ready."], ["No se pudo consultar qué está instalado.", "Could not check what is installed."], ["Marcá qué instalar.", "Tick what to install."], ["Instalando programas", "Installing programs"], ["Actualizando programas", "Updating programs"], ["winget está actualizando tus programas… no cierres esta ventana.", "winget is updating your programs… do not close this window."], ["Listo: todo lo que se podía actualizar quedó al día.", "Done: everything that could be updated is up to date."], ["Se van a actualizar todos los programas instalados que winget pueda actualizar. Puede tardar varios minutos. ¿Seguimos?", "All installed programs that winget can update will be updated. It can take several minutes. Continue?"], ["Instalando {}…", "Installing {1}…"], ["No se pudo: {}", "Could not: {1}"], ["No se pudo abrir: {}", "Could not open: {1}"], ["{n} de {n} ya instalados", "{1} of {2} already installed"], ["Terminado: {n} instalados, {n} con problemas.", "Finished: {1} installed, {2} with problems."], ["⚠ Windows lo deshizo: desactivalo y volvé a activarlo", "⚠ Windows undid it: turn it off and on again"],  ["⚠ Windows lo deshizo: desactivalo y volvé a activarlo", "⚠ Windows undid it: turn it off and on again"], ["Ya activo en tu sistema", "Already active on your system"], ["No disponible: {}", "Not available: {1}"],
  ["Todo lo de esta sección se puede deshacer: apagá el interruptor o usá «Revertir los de esta sección». Cada cambio queda anotado en", "Everything in this section can be undone: turn the switch off or use «Revert the ones in this section». Every change is recorded in"],
  ["requiere que Windows esté en un SSD", "requires Windows to be on an SSD"], ["requiere 16 GB de RAM o más", "requires 16 GB of RAM or more"],
  ["pensado para PC de escritorio", "meant for desktop PCs"], ["pensado para portátiles", "meant for laptops"],
  ["requiere una GPU AMD", "requires an AMD GPU"], ["requiere una GPU NVIDIA", "requires an NVIDIA GPU"], ["requiere una GPU Intel", "requires an Intel GPU"],
  ["{n} ajustes", "{1} tweaks"],
  ["{}: se aplicarán {n} ajustes compatibles con tu equipo.\nSe creará antes un punto de restauración y podrás revertir cada ajuste.\n\n¿Continuar?", "{1}: {2} tweaks compatible with your PC will be applied.\nA restore point will be created first and you will be able to revert each tweak.\n\nContinue?"],
  ["{}\n\n{}\n\n¿Quieres aplicarlo de todos modos?", "{1}\n\n{2}\n\nDo you want to apply it anyway?"], ["Este ajuste es avanzado.", "This tweak is advanced."],
  ["⚠ ANTI-CHEAT\n\n«{}» desactiva: {}.\nLos anti-cheat de kernel lo exigen: si lo desactivás probablemente no puedas jugar partidas competitivas hasta volver a activarlo.\n\nPrimera confirmación: ¿querés continuar?", "⚠ ANTI-CHEAT\n\n«{1}» turns off: {2}.\nKernel anti-cheats require it: if you turn it off you probably will not be able to play competitive matches until you turn it back on.\n\nFirst confirmation: do you want to continue?"],
  ["Segunda confirmación.\nPara desactivar «{}» escribí DESACTIVAR:", "Second confirmation.\nTo turn off «{1}» type DESACTIVAR:"],
  ["Cancelado: no se cambió nada.", "Cancelled: nothing was changed."], ["Terminado: {n} correctos, {n} con error.", "Done: {1} succeeded, {2} with errors."],
  ["Reinicia el equipo para completar algunos cambios.", "Restart the computer to complete some changes."], [" (reinicia para completar)", " (restart to complete)"],
  ["Ese perfil ya está aplicado en tu equipo.", "That profile is already applied on your PC."], ["No hay nada para ", "There is nothing to "],
  ["Sin perfiles.", "No profiles."],
  /* ---- limpieza ---- */
  ["Libera memoria RAM y espacio en disco: borra archivos temporales y cachés que Windows y los juegos vuelven a crear solos.", "Frees up space by deleting temporary files and caches that Windows and games recreate on their own."],
  ["Calculando…", "Calculating…"], ["{n} archivos", "{1} files"], ["Limpiar lo seleccionado", "Clean the selection"],
  ["Se pueden liberar hasta {}", "Up to {1} can be freed"], ["Selecciona qué limpiar.", "Select what to clean."], ["Limpiando", "Cleaning"], ["Borrando archivos…", "Deleting files…"],
  ["{}: {} liberados", "{1}: {2} freed"], [" ({n} en uso, omitidos)", " ({1} in use, skipped)"], ["Total liberado: {}", "Total freed: {1}"],
  ["Limpieza profunda", "Deep cleanup"],
  ["Cachés de navegadores y apps. No borra contraseñas, cookies ni historial; la primera carga después puede ser un poco más lenta. Vienen desmarcadas.", "Browser and app caches. It does not delete passwords, cookies or history; the first load afterwards may be a bit slower. They come unchecked."],
  /* ---- inicio de windows ---- */
  ["Programas que arrancan con tu sesión. Desactiva los que no necesites: no se borran, puedes volver a activarlos.", "Programs that start with your session. Turn off the ones you do not need: they are not deleted, you can turn them back on."],
  ["No se encontraron programas de inicio en el registro.", "No startup programs were found in the registry."], ["No se pudo cambiar: ", "Could not change: "],
  /* ---- historial ---- */
  ["Todos los cambios aplicados y revertidos, del más reciente al más antiguo. Desde acá podés deshacer lo que siga aplicado.", "All applied and reverted changes, from newest to oldest. From here you can undo whatever is still applied."],
  ["Borrar historial", "Clear history"], ["Todavía no hay cambios registrados.", "No changes recorded yet."], ["Aplicado", "Applied"], ["Revertido", "Reverted"], ["Deshacer", "Undo"],
  ["¿Borrar el historial? Los ajustes aplicados no se modifican.", "Clear the history? Applied tweaks are not changed."], ["Revertido.", "Reverted."], ["No se pudo: {}", "Failed: {1}"], ["falló", "failed"],
  ["Abrir web oficial", "Open official website"],
  ["Podés descargarlo de la web oficial con el botón de su fila.", "You can download it from the official website with the button in its row."],
  ["Por qué hace falta RTSS", "Why RTSS is needed"],
  ["Eso lo hace RTSS (RivaTuner Statistics Server), que se mete dentro del juego, y deja los números en una memoria compartida de Windows.", "That is done by RTSS (RivaTuner Statistics Server), which goes inside the game and leaves the numbers in a Windows shared memory."],
  ["Los FPS salen siempre de RTSS. Es liviano, queda en segundo plano y no hace falta MSI Afterburner. El Setup lo instala por vos y el overlay lo abre solo.", "FPS always come from RTSS. It is lightweight, stays in the background and MSI Afterburner is not needed. Setup installs it for you and the overlay opens it on its own."],
  ["La temperatura del procesador sale de Core Temp, que también instala el Setup y el overlay abre solo. Windows no la entrega sin un driver, y OverdriveTuner no instala drivers para no chocar con los anti-cheat. Core Temp solo la publica y OverdriveTuner la lee.", "The processor temperature comes from Core Temp, which Setup also installs and the overlay opens on its own. Windows does not provide it without a driver, and OverdriveTuner does not install drivers so as not to clash with anti-cheat. Core Temp only publishes it and OverdriveTuner reads it."],
  ["Instalá RTSS y Core Temp.", "Install RTSS and Core Temp."],
  ["El Setup y el Parche tienen una casilla para cada uno. También están en «Instalar programas».", "Setup and the Patch have a checkbox for each. They are also in «Install programs»."],
  ["Dejá el overlay activo.", "Keep the overlay active."],
  ["Al abrirse, abre RTSS y Core Temp en segundo plano si estaban cerrados.", "When it starts, it opens RTSS and Core Temp in the background if they were closed."],
  ["RTSS tiene que estar abierto antes de que arranque el juego. El overlay aparece solo, con los FPS y las temperaturas.", "RTSS has to be open before the game starts. The overlay appears on its own, with FPS and temperatures."],
  ["RTSS y Core Temp están abiertos antes de arrancar el juego.", "RTSS and Core Temp are open before the game starts."],
  ["RTSS está cerrado o se abrió con el juego ya en marcha: reiniciá el juego.", "RTSS is closed or opened after the game was already running: restart the game."],
  ["Core Temp no está instalado: el procesador no muestra temperatura.", "Core Temp is not installed: the processor shows no temperature."],
  ["Se basa en pequeños ajustes de uso diario del explorador y de Windows: menos molestias, sin tocar el rendimiento.", "Small everyday tweaks to Explorer and Windows: fewer annoyances, without touching performance."],
  ["Se basa en controlar cuándo y cómo Windows Update instala cosas, para que no te interrumpa mientras jugás.", "It controls when and how Windows Update installs things, so it does not interrupt you while you play."],
  ["La temperatura del CPU sale de Core Temp, que el overlay abre solo en segundo plano (Windows no la entrega sin un driver y OverdriveTuner no instala drivers para no chocar con los anti-cheat de kernel).", "The CPU temperature comes from Core Temp, which the overlay opens on its own in the background (Windows does not provide it without a driver and OverdriveTuner does not install drivers so as not to clash with kernel anti-cheat)."],
  ["Una guía corta: los FPS salen de RTSS y la temperatura del procesador de Core Temp. El Setup instala los dos y el overlay los abre solo.", "A short guide: FPS come from RTSS and the processor temperature from Core Temp. Setup installs both and the overlay opens them on its own."],
  ["Cómo se miden los FPS", "How FPS are measured"],
  ["Programas por categoría", "Programs by category"],
  ["Cada fila es un programa. «Instalado» te dice que ya lo tenés. Los que dependen de tu hardware (AMD, NVIDIA, Intel) solo aparecen si tu PC los usa.", "Each row is a program. «Installed» means you already have it. The ones that depend on your hardware (AMD, NVIDIA, Intel) only appear if your PC uses them."],
  ["Marcar lo que querés", "Tick what you want"],
  ["Tildá los programas que querés instalar. Hace falta abrir OverdriveTuner como administrador.", "Tick the programs you want to install. OverdriveTuner has to run as administrator."],
  ["Instalar seleccionados", "Install selected"],
  ["Los instala uno por uno con winget, el administrador de paquetes de Windows 11, y muestra el resultado de cada uno.", "It installs them one by one with winget, the Windows 11 package manager, and shows the result of each."],
  ["Actualizar programas", "Update programs"],
  ["Pide a winget que actualice todo lo que tengas instalado y se pueda actualizar. Puede tardar varios minutos.", "Asks winget to update everything you have installed that can be updated. It may take several minutes."],
  ["Si winget no puede instalar un programa", "If winget cannot install a program"],
  ["Si winget no está disponible o falla con un programa, esa fila muestra el botón «Abrir web oficial»: te lleva al sitio del fabricante para que lo descargues e instales vos.", "If winget is not available or fails with a program, that row shows the «Open official website» button: it takes you to the maker's site so you can download and install it yourself."],
  ["Drivers de video", "Video drivers"],
  ["Abre la página oficial de NVIDIA, AMD o Intel. Los drivers no se instalan desde acá: los descargás y verificás vos.", "Opens the official NVIDIA, AMD or Intel page. Drivers are not installed from here: you download and check them yourself."],
  ["Programas que se cierran", "Programs that get closed"],
  ["La lista de programas que se cierran solos cuando el overlay detecta que abriste un juego, para liberar procesador y memoria. Se cierran todos sus procesos.", "The list of programs that close on their own when the overlay detects you opened a game, to free processor and memory. All their processes are closed."],
  ["Agregar uno", "Add one"],
  ["Escribí el nombre del programa, por ejemplo chrome.exe, y tocá «Agregar». Los programas de Windows y de OverdriveTuner están protegidos y no se pueden cerrar desde acá.", "Type the program name, for example chrome.exe, and press «Add». Windows and OverdriveTuner programs are protected and cannot be closed from here."],
  ["Elegir de los abiertos", "Pick from open programs"],
  ["Mide 2 segundos y te sugiere los programas abiertos que más memoria usan. Agregás con un clic.", "Measures for 2 seconds and suggests the open programs that use the most memory. Add with one click."],
  ["Launchers y anti-cheat", "Launchers and anti-cheat"],
  ["Si agregás un launcher (Steam, EA, FACEIT…) te avisamos: cerrarlo puede sacarte del juego o impedir que entres a la partida.", "If you add a launcher (Steam, EA, FACEIT…) we warn you: closing it can kick you out of the game or stop you from joining the match."],
  ["Hace falta el overlay", "The overlay is needed"],
  ["El Modo juego lo ejecuta el overlay, que es el que detecta el juego. Si no está abierto, este botón lo abre.", "Game mode is run by the overlay, which is what detects the game. If it is not open, this button opens it."],
  ["Inicio con Windows", "Start with Windows"],
  ["El overlay arranca solo al encender la PC y aparece cuando abrís un juego. Acá ves si quedó activo y podés probarlo sin reiniciar.", "The overlay starts on its own when the PC turns on and appears when you open a game. Here you see whether it is active and can test it without restarting."],
  ["No es comparable con el 1% low de otros programas: comparalo solo con otras corridas de esta app.", "It is not comparable with the 1% low of other programs: compare it only with other runs of this app."],
  ["El overlay la lee de Core Temp, que se abre solo en segundo plano si lo tenés instalado. Solo lee: no se mete en el juego. Core Temp es gratis y está en «Instalar programas».", "The overlay reads it from Core Temp, which opens on its own in the background if you have it installed. It only reads: it does not enter the game. Core Temp is free and is in «Install programs»."],
  ["Windows no la entrega en la mayoría de los procesadores. El overlay la lee de Core Temp, que se abre solo en segundo plano si lo tenés instalado. Solo lee: no se mete en el juego. Core Temp es gratis y está en «Instalar programas».", "Windows does not provide it on most processors. The overlay reads it from Core Temp, which opens on its own in the background if you have it installed. It only reads: it does not enter the game. Core Temp is free and is in “Install programs”."],
  ["Cuando el procesador o la gráfica pasan el límite, el valor se pone en rojo y el overlay muestra un aviso de texto (sin sonido). La de la gráfica funciona siempre; la del procesador solo si Core Temp está instalado.", "When the processor or graphics card passes the limit, the value turns red and the overlay shows a text warning (no sound). The graphics card one always works; the processor one only if Core Temp is installed."],
].forEach(([a, b]) => T(a, b));

/* ---------------- benchmark, sesiones y búsqueda del culpable ---------------- */
[
  ["Pegá el resultado del benchmark de CS2 (Afterburner/FPS Benchmark). Marcá en qué estado estaba el PC y compará con varias corridas: una sola no alcanza, el ruido entre corridas es de ±20 FPS.", "Paste the CS2 benchmark result (Afterburner/FPS Benchmark). Mark what state the PC was in and compare several runs: one is not enough, the noise between runs is about ±20 FPS."],
  ["Pegá acá el texto del benchmark…", "Paste the benchmark text here…"], ["Estado:", "State:"], ["ej. antes / optimizado / sin tweaks de red", "e.g. before / optimized / without network tweaks"],
  ["es el «antes de optimizar»", "is the «before optimizing» baseline"], ["Guardar corridas", "Save runs"], ["Todavía no cargaste corridas.", "You have not loaded any runs yet."],
  ["Resumen por estado", "Summary by state"], ["Estado", "State"], ["Corridas", "Runs"], ["FPS prom.", "Avg FPS"], ["Rango", "Range"], ["Desvío", "Deviation"], ["1% low", "1% low"], ["0.1% low", "0.1% low"],
  ["base", "baseline"], ["sin etiqueta", "no label"], ["Borrar todo", "Delete all"], ["¿Borrar todas las corridas guardadas?", "Delete all saved runs?"],
  ["{n} FPS", "{1} FPS"], ["1%: {n}", "1%: {1}"], ["{n} picos", "{1} spikes"],
  ["Faltan corridas: con menos de 3 por estado no se puede separar el ruido de la diferencia real. Hacé al menos 3 en cada uno, con las mismas apps abiertas.", "More runs needed: with fewer than 3 per state, noise cannot be separated from real differences. Do at least 3 of each, with the same apps open."],
  ["La diferencia está dentro del ruido normal entre corridas: no hay evidencia de que haya cambiado el rendimiento.", "The difference is within the normal noise between runs: there is no evidence that performance changed."],
  ["Mejora real: la diferencia supera el ruido.", "Real improvement: the difference exceeds the noise."],
  ["Empeora de verdad: la diferencia supera el ruido. Probá revertir lo último que aplicaste.", "Real regression: the difference exceeds the noise. Try reverting the last thing you applied."],
  ["no se encontró ningún resultado. Pega el texto completo del benchmark (con «Average framerate», «1% low framerate», etc.)", "no result was found. Paste the full benchmark text (with «Average framerate», «1% low framerate», etc.)"],
  ["Corridas medidas con el overlay", "Runs measured with the overlay"],
  ["Grabalas desde «Overlay en juego». Se miden cuadro por cuadro, así que se ven los microcortes. Importá una al benchmark para compararla.", "Record them from «In-game overlay». They are measured frame by frame, so micro-stutters show up. Import one into the benchmark to compare it."],
  ["Todavía no grabaste ninguna.", "You have not recorded any yet."],
  ["Cada barra es un tramo de cuadros; rojo = pico (microcorte), amarillo = tirón leve. {n} picos en {n} s.", "Each bar is a stretch of frames; red = spike (micro-stutter), yellow = slight hitch. {1} spikes in {2} s."],
  ["Esa sesión ya está en el benchmark.", "That session is already in the benchmark."], ["sesión no encontrada", "session not found"],
  ["Analizando microcortes…", "Analyzing micro-stutters…"],
  ["Fluidez: muy buena", "Smoothness: very good"], ["Fluidez: con tirones", "Smoothness: with hitches"], ["Fluidez: con muchos microcortes", "Smoothness: with many micro-stutters"], ["de 100", "out of 100"],
  ["{n} picos · {n} por minuto · cuadro típico {n} ms · peor cuadro {n} ms", "{1} spikes · {2} per minute · typical frame {3} ms · worst frame {4} ms"],
  [" · se repite cada ~{n} s", " · repeats every ~{1} s"],
  ["Es una estimación por patrón: indica la causa más probable, no una certeza. Confirmala con el benchmark antes y después de cada cambio.", "This is a pattern-based estimate: it points to the most likely cause, not a certainty. Confirm it with the benchmark before and after each change."],
  ["Muy pocos datos para analizar. Grabá al menos 30 segundos de partida.", "Too little data to analyze. Record at least 30 seconds of gameplay."],
  ["Prácticamente sin microcortes: la partida salió fluida.", "Practically no micro-stutters: the match ran smoothly."],
  ["Los cortes se concentran al empezar: casi seguro es carga de recursos y compilación de shaders.", "The stutters are concentrated at the start: almost certainly asset loading and shader compilation."],
  ["Jugá una partida completa antes de medir: los shaders se compilan una vez y quedan guardados.", "Play a full match before measuring: shaders are compiled once and then saved."],
  ["No limpies la caché de shaders justo antes de jugar.", "Do not clean the shader cache right before playing."],
  ["Instalá el juego en el SSD si está en el disco duro.", "Install the game on the SSD if it is on the hard drive."],
  ["Los cortes se repiten con un ritmo parejo: suele ser una tarea de fondo que despierta cada tanto.", "The stutters repeat at a steady rhythm: it is usually a background task that wakes up every so often."],
  ["Cerrá lanzadores, antivirus con escaneo, sincronización de nube y programas de RGB o monitoreo.", "Close launchers, antivirus scans, cloud sync and RGB or monitoring programs."],
  ["Mirá «Procesos que más consumen» en Diagnóstico durante una partida.", "Look at the top processes in Diagnostics during a match."],
  ["Si el intervalo ronda los 60 segundos, sospechá de Windows Update o del antivirus.", "If the interval is around 60 seconds, suspect Windows Update or the antivirus."],
  ["Los cortes vienen en ráfagas: suele pasar en zonas o momentos concretos del mapa (mucha carga de CPU, memoria de video llena o limitación térmica).", "The stutters come in bursts: it usually happens in specific areas or moments of the map (heavy CPU load, full video memory or thermal limiting)."],
  ["Probá bajar texturas y sombras para ver si la ráfaga desaparece.", "Try lowering textures and shadows to see if the bursts go away."],
  ["Revisá temperaturas y frecuencias con el overlay durante esas ráfagas.", "Check temperatures and clocks with the overlay during those bursts."],
  ["Un límite de FPS apenas por debajo de tu tasa de refresco reduce los picos.", "An FPS cap just below your refresh rate reduces the spikes."],
  ["Los cortes aparecen sin patrón claro: lo más común es un núcleo del procesador al límite o procesos de fondo.", "The stutters show up with no clear pattern: most commonly a processor core at its limit or background processes."],
  ["Cerrá navegador y overlays de terceros antes de jugar.", "Close the browser and third-party overlays before playing."],
  ["Comprobá que la memoria corra a su velocidad (DOCP/EXPO/XMP) y en dual channel.", "Check that the memory runs at its rated speed (DOCP/EXPO/XMP) and in dual channel."],
  ["Compará con el benchmark guiado antes y después de cada ajuste.", "Compare with the guided benchmark before and after each tweak."],
  ["Buscar el ajuste culpable", "Find the culprit tweak"], ["Buscar el ajuste que empeora el rendimiento", "Find the tweak that hurts performance"],
  ["de verdad", "for real"], ["Si después de optimizar rindió peor", "If performance got worse after optimizing"],
  ["Ronda {n}: ", "Round {1}: "], ["se revirtieron {n} ajustes", "{1} tweaks were reverted"], [" de {n} sospechosos. Medí con las mismas apps abiertas (podés grabar con el overlay) y contestá:", " out of {1} suspects. Measure with the same apps open (you can record with the overlay) and answer:"],
  ["El problema desapareció", "The problem went away"], ["Sigue igual", "Still the same"], ["Cancelar y volver a aplicar todo", "Cancel and re-apply everything"], ["Empezar otra búsqueda", "Start another search"],
  ["hacen falta al menos 2 ajustes aplicados para buscar cuál es el culpable", "at least 2 applied tweaks are needed to find the culprit"],
  ["Medí ahora (mismas apps abiertas que antes) y decime si el problema desapareció.", "Measure now (same apps open as before) and tell me whether the problem went away."],
  ["no hay una búsqueda en curso", "there is no search in progress"],
  ["Encontrado: este ajuste es el que empeora el rendimiento. Queda revertido; el resto se volvió a aplicar.", "Found: this tweak is the one hurting performance. It stays reverted; the rest were re-applied."],
  ["Ningún ajuste individual explica el problema (puede ser ruido, otra causa del sistema o varios ajustes a la vez). Se volvió a aplicar todo.", "No single tweak explains the problem (it may be noise, another cause on the system, or several tweaks at once). Everything was re-applied."],
  ["Búsqueda cancelada; se volvieron a aplicar los ajustes.", "Search cancelled; the tweaks were re-applied."], ["Ya hay una búsqueda en curso.", "A search is already in progress."],
].forEach(([a, b]) => T(a, b));

/* ---------------- overlay ---------------- */
[
  ["Un panel mínimo con FPS, CPU, GPU y RAM encima del juego. No se inyecta en el juego (es una ventana aparte que deja pasar los clics), por eso no molesta a los anti-cheat de kernel ni a VAC. Los atajos de teclado se cambian en Configuración.", "A minimal panel with FPS, CPU, GPU and RAM on top of the game. It is not injected into the game (it is a separate window that lets clicks through), so it does not bother kernel anti-cheats or VAC. You can change the shortcuts in Settings."],
  ["Entender los números", "Understand the numbers"], ["Vista previa", "Preview"], ["Apariencia", "Appearance"], ["Esquina", "Corner"],
  ["Arriba izquierda", "Top left"], ["Arriba derecha", "Top right"], ["Abajo izquierda", "Bottom left"], ["Abajo derecha", "Bottom right"],
  ["Diseño", "Layout"], ["Tarjeta", "Card"], ["Una línea", "Single line"], ["Tamaño", "Size"], ["Transparencia", "Transparency"], ["Qué mostrar", "What to show"],
  ["Gráfico de cuadros", "Frame graph"], ["Temperaturas", "Temperatures"], ["Juego a medir", "Game to measure"], ["Nombre del programa, por ejemplo cs2.exe", "Program name, for example cs2.exe"],
  ["Es una aproximación: el diseño real lo dibuja Windows y puede verse un poco distinto.", "This is an approximation: the real layout is drawn by Windows and may look slightly different."],
  ["Overlay abierto", "Overlay open"], ["Overlay cerrado", "Overlay closed"], ["{} detectado", "{1} detected"], ["esperando {}", "waiting for {1}"],
  ["Abrir overlay", "Open overlay"], ["Cerrar overlay", "Close overlay"], ["Abrilo para ver FPS y uso mientras jugás.", "Open it to see FPS and usage while you play."],
  ["Qué limita tus FPS:", "What limits your FPS:"], ["no disponible", "not available"],
  ["FPS {n} · 1% low {n} · CPU {n}% (hilo más cargado {n}%) · GPU {n}% {} · RAM {n} GB · temp. CPU {}", "FPS {1} · 1% low {2} · CPU {3}% (busiest thread {4}%) · GPU {5}% {6} · RAM {7} GB · CPU temp {8}"],
  ["Cerrando overlay…", "Closing overlay…"], ["Abriendo overlay…", "Opening overlay…"],
  ["Grabar una corrida", "Record a run"],
  ["Poné un nombre que describa el estado del PC (por ejemplo «antes» o «optimizado»), tocá Grabar, jugá el mismo tramo o benchmark y tocá Detener. Hacé 3 corridas por estado. Después se importan en «Benchmark».", "Enter a name that describes the PC's state (for example «before» or «optimized»), tap Record, play the same stretch or benchmark and tap Stop. Do 3 runs per state. Then import them in «Benchmark»."],
  ["antes / optimizado…", "before / optimized…"], ["Grabar", "Record"], ["Detener y guardar", "Stop and save"], ["Grabando…", "Recording…"],
  ["Guardando sesión… mirala en Benchmark.", "Saving session… see it in Benchmark."],
  ["Perfiles por juego", "Per-game profiles"], ["Agregar juego", "Add game"], ["otro juego, ej. eurotrucks2.exe", "another game, e.g. eurotrucks2.exe"],
  ["Prioridad alta al abrir", "High priority on launch"], ["Plan de energía:", "Power plan:"], ["no cambiar", "do not change"],
  ["Cerrar al abrir el juego (separados por coma):", "Close when the game opens (comma separated):"],
  ["Los programas se cierran sin guardar; nunca se cierran procesos de Windows. El plan de energía vuelve al anterior al cerrar el juego.", "Programs are closed without saving; Windows processes are never closed. The power plan returns to the previous one when the game closes."],
  ["Abrí primero el overlay.", "Open the overlay first."], ["El juego debe ser un nombre de programa como cs2.exe", "The game must be a program name like cs2.exe"],
  ["No se pudo abrir el overlay: {}", "Could not open the overlay: {1}"],
  ["Grabando: abrí el juego ({}) para que lleguen cuadros.", "Recording: open the game ({1}) so frames arrive."],
  ["La grabación fue muy corta ({n} cuadros); no se guardó.", "The recording was too short ({1} frames); it was not saved."],
  ["Sesión guardada: {n} FPS promedio, {n} picos.", "Session saved: {1} average FPS, {2} spikes."], ["No se pudo guardar la sesión: {}", "Could not save the session: {1}"],
  ["No se pudo medir los FPS: {}", "Could not measure FPS: {1}"],
  ["Faltan datos: jugá unos segundos con el overlay activo.", "Not enough data: play for a few seconds with the overlay on."],
  ["Limita la gráfica ({n}% de uso). Bajar resolución, sombras o efectos sube los FPS.", "The graphics card is the limit ({1}% usage). Lowering resolution, shadows or effects raises FPS."],
  ["Limita un solo hilo del procesador ({n}%), aunque el uso total sea de apenas {n}%. Es normal en CS2: ayudan la RAM a su velocidad (DOCP/XMP), cerrar programas de fondo y bajar efectos que cargan al procesador.", "A single processor thread is the limit ({1}%), even though total usage is only {2}%. This is normal in CS2: RAM at its rated speed (DOCP/XMP), closing background programs and lowering CPU-heavy effects help."],
  ["El procesador está saturado ({n}%). Cerrá programas de fondo.", "The processor is saturated ({1}%). Close background programs."],
  ["Ni el procesador (hilo máx. {n}%) ni la gráfica ({n}%) llegan al límite. Puede haber un tope de FPS (fps_max, V-Sync), o limita la latencia de la memoria RAM.", "Neither the processor (max thread {1}%) nor the graphics card ({2}%) reach their limit. There may be an FPS cap (fps_max, V-Sync), or RAM latency is the limit."],
  /* guía de números */
  ["Qué significa cada número", "What each number means"], ["Este es un overlay de ejemplo con valores de muestra. Usá «Siguiente» para recorrerlo.", "This is a sample overlay with example values. Use «Next» to walk through it."],
  ["FPS", "FPS"], ["Cuántos cuadros por segundo dibuja el juego ahora mismo. En un monitor de 240 Hz, lo ideal es estar cerca de 240.", "How many frames per second the game is drawing right now. On a 240 Hz monitor, the ideal is to be close to 240."],
  ["Promedio de los cuadros más lentos (el 1 % peor). Si está muy por debajo del FPS, hay tirones aunque el promedio se vea alto. No es comparable con el 1% low de Afterburner: comparalo solo con otras corridas de esta app.", "Average of the slowest frames (the worst 1%). If it is far below the FPS, there are hitches even when the average looks high. It is not comparable with Afterburner's 1% low: compare it only with other runs from this app."],
  ["Cada barra es un cuadro y la altura es lo que tardó. Verde es normal, amarillo es un tirón leve y rojo es un pico: eso es lo que sentís como microcorte.", "Each bar is a frame and its height is how long it took. Green is normal, yellow is a slight hitch and red is a spike: that is what you feel as a micro-stutter."],
  ["CPU", "CPU"], ["Uso total del procesador. Ojo: un juego como CS2 usa pocos hilos, así que el CPU puede marcar 50 % y estar limitado igual por un solo núcleo. La temperatura del CPU sale solo si MSI Afterburner está abierto (Windows no la entrega sin un driver y OverdriveTuner no instala drivers para no chocar con los anti-cheat de kernel).", "Total processor usage. Note: a game like CS2 uses few threads, so the CPU can show 50% and still be limited by a single core. The CPU temperature only appears if MSI Afterburner is open (Windows does not provide it without a driver and OverdriveTuner does not install drivers so it does not clash with kernel anti-cheats)."],
  ["GPU", "GPU"], ["Uso del motor 3D de la placa de video y su temperatura. Si pasa de 92 %, el límite lo pone la placa de video. Temperatura: verde hasta 69°, amarillo desde 70° y rojo desde 85°.", "Usage of the graphics card's 3D engine and its temperature. Above 92%, the graphics card is the limit. Temperature: green up to 69°, yellow from 70° and red from 85°."],
  ["VRAM", "VRAM"], ["Memoria de video en uso. Si se llena, aparecen tirones fuertes: bajá texturas o resolución.", "Video memory in use. If it fills up, heavy hitches appear: lower textures or resolution."],
  ["RAM", "RAM"], ["Memoria del sistema en uso. Si se acerca a tu total, Windows empieza a usar el disco y se nota.", "System memory in use. If it gets close to your total, Windows starts using the disk and you notice it."],
  ["Qué limita tus FPS", "What limits your FPS"], ["En la pantalla de Overlay, la app combina estos números y te dice si el límite es la placa de video, un hilo del CPU u otra cosa, para saber qué conviene ajustar.", "On the Overlay screen, the app combines these numbers and tells you whether the limit is the graphics card, a CPU thread or something else, so you know what to adjust."],
].forEach(([a, b]) => T(a, b));

/* ---------------- diagnóstico, herramientas y guías ---------------- */
[
  ["Revisa lo que más causa microcortes y falta de fluidez, sobre todo en monitores de alta frecuencia. Nada se cambia solo.", "Checks what most often causes micro-stutters and lack of smoothness, especially on high refresh rate monitors. Nothing is changed on its own."],
  ["Revisando…", "Checking…"], ["{n} problema(s) importante(s)", "{1} important problem(s)"], [" · {n} aviso(s)", " · {1} warning(s)"], ["Sin problemas importantes", "No important problems"],
  [" · monitor objetivo: {n} Hz", " · target monitor: {1} Hz"], ["Frecuencia de tu monitor (Hz):", "Your monitor's refresh rate (Hz):"], ["Volver a revisar", "Check again"],
  ["Qué está usando tu procesador", "What is using your processor"], ["Medir ahora (2 segundos)", "Measure now (2 seconds)"], ["Midiendo…", "Measuring…"], ["Nada usa CPU de forma notable.", "Nothing uses a notable amount of CPU."],
  ["{n}% de un núcleo", "{1}% of one core"], ["{n} MB", "{1} MB"], ["protegido", "protected"], ["Cerrar «{}»? Perdés lo que no hayas guardado en ese programa.", "Close «{1}»? You lose anything unsaved in that program."], ["Cerrado: {}", "Closed: {1}"],
  ["Herramientas", "Tools"], ["Probar red y ping (15 s)", "Test network and ping (15 s)"], ["Probar DNS", "Test DNS"], ["Quién usa la red", "Who is using the network"], ["Estado de los discos", "Disk status"], ["Revisar video de CS2", "Check CS2 video settings"], ["Descargar informe", "Download report"],
  ["Midiendo (unos 15 segundos)…", "Measuring (about 15 seconds)…"], ["Midiendo servidores DNS (unos 10 segundos)…", "Measuring DNS servers (about 10 seconds)…"], ["Leyendo conexiones…", "Reading connections…"], ["Leyendo discos…", "Reading disks…"], ["Buscando cs2_video.txt…", "Looking for cs2_video.txt…"],
  ["bien", "good"], ["aviso", "warning"], ["problema", "problem"], ["info", "info"], ["sin datos", "no data"],
  ["Promedio {n} ms · mín {n} · máx {n} · variación {n} ms · pérdida {n}%. ", "Average {1} ms · min {2} · max {3} · jitter {4} ms · loss {5}%. "],
  ["Pierde paquetes: se nota como tirones y desincronización. Probá con cable de red en vez de Wi-Fi.", "Packet loss: you notice it as hitches and desync. Try a network cable instead of Wi-Fi."],
  ["Conexión irregular: hay pérdida, ping alto o variaciones.", "Unstable connection: there is loss, high ping or variation."], ["Estable.", "Stable."],
  ["No se pudo medir (sin conexión o ping bloqueado).", "Could not measure (no connection or ping blocked)."],
  ["Windows no devolvió información de los discos.", "Windows did not return any disk information."],
  ["Mide cuánto tarda cada servidor en responder desde tu conexión. Cambiar de DNS acelera la resolución de nombres (conectar a servidores), no baja el ping dentro de la partida. Se guarda tu configuración anterior.", "Measures how long each server takes to respond from your connection. Changing DNS speeds up name resolution (connecting to servers); it does not lower in-match ping. Your previous configuration is saved."],
  ["más rápido", "fastest"], ["en uso", "in use"], ["No respondió desde tu conexión.", "It did not respond from your connection."], ["Respuesta típica: ", "Typical response: "], [" · {n} consulta(s) fallaron", " · {1} quer(ies) failed"],
  ["Usar este DNS", "Use this DNS"], ["Volver al anterior", "Go back to the previous one"], ["DNS cambiado.", "DNS changed."], ["DNS restaurado.", "DNS restored."], ["No se pudo cambiar el DNS", "Could not change the DNS"],
  ["Programas con conexiones abiertas. La columna de actividad es una estimación (incluye otras entradas y salidas del programa): sirve para encontrar quién usa tu red mientras jugás.", "Programs with open connections. The activity column is an estimate (it includes other input/output of the program): it helps you find who is using your network while you play."],
  ["Programa", "Program"], ["Actividad", "Activity"], ["Se conecta a", "Connects to"], ["{n} KB/s", "{1} KB/s"], ["No se encontraron programas con conexiones abiertas.", "No programs with open connections were found."],
  ["{n} GB libres de {n} GB", "{1} GB free of {2} GB"],
  ["{} · desgaste {n} %", "{1} · wear {2} %"], ["{n} h encendido", "{1} h powered on"], ["TRIM activo", "TRIM on"], ["TRIM desactivado", "TRIM off"], ["{n} °C", "{1} °C"],
  ["Sin problemas.", "No problems."],
  ["Windows reporta una advertencia en este disco. Hacé una copia de lo importante.", "Windows reports a warning on this disk. Back up what is important."],
  ["Windows reporta que este disco puede fallar. Copiá tus archivos ya.", "Windows reports that this disk may fail. Copy your files now."],
  ["El SSD usó casi toda su vida útil. Hacé copia de tus archivos y planificá reemplazarlo.", "The SSD has used almost all of its lifespan. Back up your files and plan to replace it."],
  ["El SSD ya usó buena parte de su vida útil.", "The SSD has already used a good part of its lifespan."],
  ["El disco registró errores de lectura o escritura sin corregir. Hacé una copia de lo importante.", "The disk logged uncorrected read or write errors. Back up what is important."],
  ["El disco está muy caliente. Revisá la ventilación de la caja.", "The disk is very hot. Check the case ventilation."],
  ["La unidad {}: tiene menos del 10 % libre. Con poco espacio libre los juegos cargan peor y el SSD se desgasta más.", "Drive {1}: has less than 10% free. With little free space games load worse and the SSD wears faster."],
  ["TRIM está desactivado en Windows: activalo con «fsutil behavior set DisableDeleteNotify 0» desde un CMD de administrador.", "TRIM is turned off in Windows: turn it on with «fsutil behavior set DisableDeleteNotify 0» from an administrator CMD."],
  ["Ideal para instalar los juegos y Windows.", "Ideal for installing games and Windows."],
  ["Disco mecánico: mejor para archivos y copias; los juegos cargan más lento y pueden dar tirones.", "Mechanical disk: better for files and backups; games load slower and can hitch."],
  ["V-Sync activado: agrega latencia y limita los FPS. Para competir conviene desactivarlo.", "V-Sync on: it adds latency and caps FPS. For competitive play it is better off."],
  ["Frecuencia en el juego", "Refresh rate in the game"],
  ["El juego está configurado a {n} Hz y tu monitor es de {n} Hz. Cambialo en Configuración > Video.", "The game is set to {1} Hz and your monitor is {2} Hz. Change it in Settings > Video."],
  ["MSAA 4x o más cuesta bastante rendimiento. CMAA2 o 2x suelen dar buena imagen con menos costo.", "MSAA 4x or higher costs a lot of performance. CMAA2 or 2x usually give a good image at a lower cost."],
  ["Modo de pantalla", "Display mode"], ["En ventana puede sumar algo de latencia; probá pantalla completa.", "Windowed mode can add some latency; try fullscreen."],
  ["No encontré cs2_video.txt. Abrí CS2 al menos una vez (y cerralo) para que Steam lo cree.", "I could not find cs2_video.txt. Open CS2 at least once (and close it) so Steam creates it."],
  ["No se pudo leer el archivo: {}", "Could not read the file: {1}"],
  ["Solo se revisan las opciones que OverdriveTuner reconoce; el resto del archivo no se toca.", "Only the options OverdriveTuner recognizes are checked; the rest of the file is not touched."],
  /* diagnóstico */
  ["Gráfica que maneja el monitor", "Graphics card driving the monitor"], ["No se pudo determinar qué gráfica tiene el monitor conectado.", "Could not determine which graphics card the monitor is connected to."],
  ["Tu monitor está conectado a la gráfica integrada ({}), no a la dedicada. Los juegos se dibujan en la placa de video pero la imagen pasa por la integrada: eso agrega latencia y microcortes y puede limitar los Hz. Pasá el cable del monitor a una salida DisplayPort/HDMI de la placa de video (en la parte baja de la parte trasera del PC, no la de la placa madre).", "Your monitor is connected to the integrated graphics ({1}), not the dedicated card. Games are drawn on the graphics card but the image goes through the integrated one: that adds latency and micro-stutters and can limit the Hz. Move the monitor cable to a DisplayPort/HDMI output on the graphics card (the lower part of the back of the PC, not the motherboard one)."],
  ["El monitor está conectado a {}.", "The monitor is connected to {1}."], ["Frecuencia del monitor", "Monitor refresh rate"],
  ["Windows está usando {n} Hz y tu monitor es de {n} Hz. Con menos Hz la imagen se ve menos fluida aunque el juego dé más FPS. Cambialo en Pantalla avanzada > Frecuencia de actualización. Si no aparece {n} Hz, revisá el cable (DisplayPort o HDMI 2.0 o superior) y que el monitor tenga el modo activado en su menú.", "Windows is using {1} Hz and your monitor is {2} Hz. With fewer Hz the picture looks less smooth even if the game gives more FPS. Change it in Advanced display > Refresh rate. If {3} Hz does not appear, check the cable (DisplayPort or HDMI 2.0 or higher) and that the monitor has the mode enabled in its menu."],
  ["Abrir configuración de pantalla", "Open display settings"], ["Windows está usando {n} Hz.", "Windows is using {1} Hz."],
  ["Velocidad de la memoria", "Memory speed"], ["Tu RAM funciona a {n} MT/s pero está preparada para {n} MT/s. Activá el perfil {} en la BIOS: es de lo que más FPS parejos da en CS2.", "Your RAM runs at {1} MT/s but is rated for {2} MT/s. Turn on the {3} profile in the BIOS: it is among the things that give the most consistent FPS in CS2."],
  ["La RAM funciona a {n} MT/s.", "The RAM runs at {1} MT/s."], ["Canales de memoria", "Memory channels"],
  ["Hay un solo módulo de RAM: funciona en canal simple, que rinde bastante menos. Con dos módulos iguales en las ranuras que indica el manual de tu placa (suelen ser la 2.ª y la 4.ª contando desde el procesador) pasa a dual channel.", "There is only one RAM module: it runs in single channel, which performs a lot worse. With two identical modules in the slots your motherboard manual indicates (usually the 2nd and 4th counting from the processor) it becomes dual channel."],
  ["{n} módulos. Para que sea dual channel deben estar en las ranuras que indica el manual de la placa (normalmente la 2.ª y la 4.ª contando desde el procesador).", "{1} modules. To be dual channel they must be in the slots the motherboard manual indicates (normally the 2nd and 4th counting from the processor)."],
  ["Plan de energía", "Power plan"], ["Plan activo: {}", "Active plan: {1}"], ["El plan activo es «{}», que limita el procesador. Cambialo a Equilibrado o Alto rendimiento.", "The active plan is «{1}», which limits the processor. Change it to Balanced or High performance."],
  ["Grabación en segundo plano (Game DVR)", "Background recording (Game DVR)"], ["Windows puede estar grabando en segundo plano. Se desactiva en «FPS y juegos».", "Windows may be recording in the background. Turn it off in «FPS and games»."],
  ["Modo Juego de Windows", "Windows Game Mode"], ["El Modo Juego está desactivado. Se activa en «FPS y juegos».", "Game Mode is off. Turn it on in «FPS and games»."],
  ["Hay {n} monitores activos. Si el segundo tiene otra frecuencia (por ejemplo 60 Hz), el escritorio de Windows puede entrecortar los juegos en ventana o sin bordes. Jugá en pantalla completa o apagá el segundo monitor al jugar.", "There are {1} active monitors. If the second one has a different refresh rate (for example 60 Hz), the Windows desktop can make windowed or borderless games stutter. Play in fullscreen or turn off the second monitor while playing."],
  ["Programación de GPU acelerada por hardware", "Hardware-accelerated GPU scheduling"], ["Está {}. En algunos juegos mejora la fluidez y en otros la empeora: si la cambiás, medilo con el benchmark.", "It is {1}. In some games it improves smoothness and in others it makes it worse: if you change it, measure it with the benchmark."],
  ["activada", "on"], ["desactivada", "off"],
  ["Integridad de memoria (HVCI)", "Memory integrity (HVCI)"], ["IOMMU / protección DMA (se activa en la BIOS: IOMMU o SVM)", "IOMMU / DMA protection (turned on in the BIOS: IOMMU or SVM)"],
  ["Windows tiene actualizaciones pendientes: reiniciá para completarlas", "Windows has pending updates: restart to complete them"],
  ["Requisitos de anti-cheat de kernel", "Kernel anti-cheat requirements"], ["No están activos: {}. Los anti-cheat de kernel los piden para jugar. OverdriveTuner nunca los desactiva sin doble confirmación.", "Not active: {1}. Kernel anti-cheats require them to play. OverdriveTuner never turns them off without a double confirmation."],
  ["Secure Boot, TPM e integridad de memoria están activos. OverdriveTuner no los toca.", "Secure Boot, TPM and memory integrity are active. OverdriveTuner does not touch them."],
  /* guías */
  ["Guía de BIOS", "BIOS guide"], ["Guía de BIOS · {}", "BIOS guide · {1}"],
  ["Armada según tu procesador y tu placa de video detectados; los nombres exactos de cada opción dependen del fabricante de tu placa madre.", "Built from your detected processor and graphics card; the exact names of each option depend on your motherboard's manufacturer."],
  ["DOCP / EXPO (memoria)", "DOCP / EXPO (memory)"], ["XMP (memoria)", "XMP (memory)"], ["XMP / DOCP (memoria)", "XMP / DOCP (memory)"],
  ["Activá el perfil de memoria en la BIOS (en placas AMD se llama DOCP o EXPO). Sin esto la RAM corre más lenta de lo que compraste.", "Turn on the memory profile in the BIOS (on AMD boards it is called DOCP or EXPO). Without this the RAM runs slower than what you paid for."],
  ["Activá el perfil de memoria en la BIOS (en placas Intel se llama XMP). Sin esto la RAM corre más lenta de lo que compraste.", "Turn on the memory profile in the BIOS (on Intel boards it is called XMP). Without this the RAM runs slower than what you paid for."],
  ["Activá el perfil de memoria en la BIOS (se llama XMP en Intel y DOCP o EXPO en AMD). Sin esto la RAM corre más lenta de lo que compraste.", "Turn on the memory profile in the BIOS (it is called XMP on Intel and DOCP or EXPO on AMD). Without this the RAM runs slower than what you paid for."],
  ["Resizable BAR", "Resizable BAR"],
  ["Con una placa de video dedicada suele dar una mejora chica pero gratis. En la BIOS: Above 4G Decoding en Enabled y Re-Size BAR Support en Enabled. Puede requerir actualizar la BIOS; hacelo solo si hace falta y siguiendo el manual.", "With a dedicated graphics card it usually gives a small but free improvement. In the BIOS: Above 4G Decoding set to Enabled and Re-Size BAR Support set to Enabled. It may require updating the BIOS; only do it if needed and following the manual."],
  ["Con una placa de video dedicada suele dar una mejora chica pero gratis. En la BIOS: Above 4G Decoding en Enabled y Re-Size BAR Support (en AMD también se llama Smart Access Memory) en Enabled. Puede requerir actualizar la BIOS; hacelo solo si hace falta y siguiendo el manual.", "With a dedicated graphics card it usually gives a small but free improvement. In the BIOS: Above 4G Decoding set to Enabled and Re-Size BAR Support (on AMD it is also called Smart Access Memory) set to Enabled. It may require updating the BIOS; only do it if needed and following the manual."],
  ["IOMMU y SVM", "IOMMU and SVM"], ["VT-d y VT-x", "VT-d and VT-x"], ["Virtualización / IOMMU", "Virtualization / IOMMU"],
  ["Los anti-cheat de kernel piden IOMMU: en la BIOS aparece como «IOMMU» o «SVM Mode». Dejalo activado.", "Kernel anti-cheats require IOMMU: in the BIOS it appears as «IOMMU» or «SVM Mode». Leave it on."],
  ["Los anti-cheat de kernel piden IOMMU: en la BIOS de Intel se llama «VT-d» (y «VT-x» para virtualización). Dejalos activados.", "Kernel anti-cheats require IOMMU: in Intel BIOS it is called «VT-d» (and «VT-x» for virtualization). Leave them on."],
  ["Los anti-cheat de kernel piden IOMMU: en la BIOS se llama «IOMMU» o «SVM» (AMD) o «VT-d» (Intel). Dejalo activado.", "Kernel anti-cheats require IOMMU: in the BIOS it is called «IOMMU» or «SVM» (AMD) or «VT-d» (Intel). Leave it on."],
  ["Secure Boot y TPM", "Secure Boot and TPM"],
  ["Los anti-cheat de kernel los exigen. El TPM en la BIOS se llama «fTPM» o «AMD CPU fTPM»: dejalo activado.", "Kernel anti-cheats require them. The TPM in the BIOS is called «fTPM» or «AMD CPU fTPM»: leave it on."],
  ["Los anti-cheat de kernel los exigen. El TPM en la BIOS se llama «PTT» (Intel Platform Trust Technology): dejalo activado.", "Kernel anti-cheats require them. The TPM in the BIOS is called «PTT» (Intel Platform Trust Technology): leave it on."],
  ["Los anti-cheat de kernel los exigen. El TPM en la BIOS se llama «fTPM» (AMD) o «PTT» (Intel): dejalo activado.", "Kernel anti-cheats require them. The TPM in the BIOS is called «fTPM» (AMD) or «PTT» (Intel): leave it on."],
  ["Antes de tocar la BIOS", "Before touching the BIOS"],
  ["Anotá cada cambio y hacelo de a uno. Si el PC no arranca, la opción «Load Optimized Defaults» lo deja como venía. Los nombres exactos cambian según el fabricante de la placa.", "Write down each change and do them one at a time. If the PC does not boot, the «Load Optimized Defaults» option puts it back as it was. The exact names change depending on the motherboard manufacturer."],
  ["Guía para jugar fluido", "Guide to playing smoothly"], ["CS2 (dentro del juego)", "CS2 (in the game)"],
  ["Multicore rendering: activado · V-Sync: desactivado · Sombras, efectos y sombreado: bajos · Oclusión ambiental: desactivada · FSR: desactivado (al estar limitado por CPU no suma). Es un juego que depende sobre todo del procesador y la memoria, no de la gráfica.", "Multicore rendering: on · V-Sync: off · Shadows, effects and shading: low · Ambient occlusion: off · FSR: off (since you are CPU-limited it adds nothing). It is a game that depends mostly on the processor and memory, not the graphics card."],
  ["Límite de FPS (probar con el benchmark)", "FPS cap (test with the benchmark)"],
  ["Poner fps_max apenas por debajo de 240 (por ejemplo 237) y, si el monitor tiene FreeSync, activarlo, puede evitar cortes. Es una idea para probar, no una garantía: medilo con el benchmark antes y después.", "Setting fps_max just below 240 (for example 237) and, if the monitor has FreeSync, turning it on, can avoid stutters. It is an idea to try, not a guarantee: measure it with the benchmark before and after."],
  ["Driver de AMD", "AMD driver"], ["NO actives Anti-Lag+ en CS2: puede ser detectado como modificación y causar baneos VAC. Si querés probar, usá solo la opción Anti-Lag común o dejalo apagado.", "Do NOT turn on Anti-Lag+ in CS2: it can be detected as a modification and cause VAC bans. If you want to try, use only the regular Anti-Lag option or leave it off."],
  ["Si el diagnóstico dice que la RAM no corre a su velocidad nominal, activar DOCP/EXPO en la BIOS es de lo que más ayuda.", "If the diagnostics say the RAM does not run at its rated speed, turning on DOCP/EXPO in the BIOS is one of the things that helps most."],
  ["Si el diagnóstico dice que la RAM no corre a su velocidad nominal, activar XMP en la BIOS es de lo que más ayuda.", "If the diagnostics say the RAM does not run at its rated speed, turning on XMP in the BIOS is one of the things that helps most."],
  ["Si el diagnóstico dice que la RAM no corre a su velocidad nominal, activar XMP/DOCP en la BIOS es de lo que más ayuda.", "If the diagnostics say the RAM does not run at its rated speed, turning on XMP/DOCP in the BIOS is one of the things that helps most."],
  ["Segundo plano", "Background"], ["Cerrá navegadores, launchers y overlays antes de jugar. Abajo ves qué proceso usa más CPU.", "Close browsers, launchers and overlays before playing. Below you can see which process uses the most CPU."],
  ["Anti-cheat de kernel", "Kernel anti-cheat"], ["OverdriveTuner no toca Secure Boot, TPM, IOMMU, integridad de memoria ni las actualizaciones de seguridad. Lo único que las afecta está en «Avanzado» y pide doble confirmación.", "OverdriveTuner does not touch Secure Boot, TPM, IOMMU, memory integrity or security updates. The only things that affect them are in «Advanced» and ask for double confirmation."],
  ["Procesos que más consumen", "Top processes"], ["el proceso ya no existe", "the process no longer exists"],
  ["«{}» es un proceso del sistema o del juego y no se cierra desde acá", "«{1}» is a system or game process and cannot be closed from here"], ["«{}» no se puede cerrar desde acá", "«{1}» cannot be closed from here"],
  ["proceso no permitido", "process not allowed"], ["el PID cambió de programa; actualizá la lista", "the PID now belongs to another program; refresh the list"], ["es un proceso de Windows y no se cierra desde acá", "it is a Windows process and cannot be closed from here"],
  ["{} — monitor conectado acá", "{1} — monitor connected here"],
].forEach(([a, b]) => T(a, b));

/* ---------------- motor: aplica el idioma al DOM ---------------- */
let UILANG = "es";
window.UILANG = "es";
const origText = new WeakMap();   // nodo de texto -> {es, en}
const origAttr = new WeakMap();   // elemento -> {attr: {es, en}}
const ATTRS = ["placeholder", "title", "aria-label", "alt"];
let obs = null, applying = false;

function browserLang() {
  const l = (navigator.language || "es").toLowerCase();
  return l.startsWith("es") ? "es" : "en";
}
function cachedLang() { try { return (window.BOOT && window.BOOT.lang) || localStorage.getItem("otLang") || ""; } catch (e) { return ""; } }
function effectiveLang(settingsLang) { return settingsLang || cachedLang() || browserLang(); }

function trNode(n) {
  const rec = origText.get(n);
  if (rec && n.data === rec.en) return;
  const es = n.data, en = UILANG === "en" ? trText(es) : es;
  if (UILANG === "en" && en !== es) { origText.set(n, { es, en }); n.data = en; }
  else if (rec) origText.delete(n);
}
function trAttrs(el) {
  for (const a of ATTRS) {
    if (!el.hasAttribute || !el.hasAttribute(a)) continue;
    const v = el.getAttribute(a);
    let rec = origAttr.get(el); const cur = rec && rec[a];
    if (cur && v === cur.en) continue;
    const en = UILANG === "en" ? trText(v) : v;
    if (UILANG === "en" && en !== v) { if (!rec) { rec = {}; origAttr.set(el, rec); } rec[a] = { es: v, en }; el.setAttribute(a, en); }
    else if (cur) delete rec[a];
  }
}
function walk(root) {
  if (!root) return;
  if (root.nodeType === 3) { trNode(root); return; }
  if (root.nodeType !== 1) return;
  const tag = root.tagName;
  if (tag === "SCRIPT" || tag === "STYLE") return;
  trAttrs(root);
  const w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT, {
    acceptNode(n) { return n.nodeType === 1 && (n.tagName === "SCRIPT" || n.tagName === "STYLE") ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT; }
  });
  let n;
  while ((n = w.nextNode())) { if (n.nodeType === 3) trNode(n); else trAttrs(n); }
}
function restoreAll() {
  const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT);
  let n;
  while ((n = w.nextNode())) {
    if (n.nodeType === 3) { const r = origText.get(n); if (r && n.data === r.en) n.data = r.es; origText.delete(n); }
    else { const r = origAttr.get(n); if (r) { for (const a in r) if (n.getAttribute(a) === r[a].en) n.setAttribute(a, r[a].es); origAttr.delete(n); } }
  }
}
function startObserver() {
  if (obs) return;
  obs = new MutationObserver(list => {
    if (applying || UILANG !== "en") return;
    applying = true;
    try { for (const m of list) { if (m.type === "characterData") trNode(m.target); else if (m.type === "attributes") trAttrs(m.target); else m.addedNodes.forEach(walk); } }
    finally { applying = false; obs.takeRecords(); }
  });
  obs.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ATTRS });
}
function applyLang(lang) {
  UILANG = lang === "en" ? "en" : "es"; window.UILANG = UILANG;
  document.documentElement.lang = UILANG;
  document.title = "OverdriveTuner";
  try { localStorage.setItem("otLang", UILANG); } catch (e) { /* sin almacenamiento */ }
  startObserver();
  applying = true;
  try { if (UILANG === "en") walk(document.body); else restoreAll(); } finally { applying = false; obs.takeRecords(); }
}

/* diálogos nativos */
function trDialog(s) { return UILANG === "en" ? String(s).split("\n").map(l => trText(l)).join("\n") : s; }
const _confirm = window.confirm.bind(window), _prompt = window.prompt.bind(window), _alert = window.alert.bind(window);
window.confirm = m => _confirm(trDialog(m));
window.prompt = (m, d) => _prompt(trDialog(m), d);
window.alert = m => _alert(trDialog(m));

/* ---------------- recorridos, categorías y Configuración ---------------- */
[
  ["Esta categoría se basa en sacarle trabajo innecesario al sistema mientras jugás: prioridades, modo juego y planificación de CPU.", "This category is about taking unnecessary work off the system while you play: priorities, game mode and CPU scheduling."],
  ["Se basa en reducir el retraso entre mover el mouse o apretar una tecla y verlo en pantalla: temporizadores, aceleración y cola de entrada.", "It is about reducing the delay between moving the mouse or pressing a key and seeing it on screen: timers, acceleration and the input queue."],
  ["Se basa en cómo Windows y el driver reparten el trabajo de la placa de video: programación por hardware, DWM y modos de presentación.", "It is about how Windows and the driver share the graphics card's work: hardware scheduling, DWM and presentation modes."],
  ["Se basa en que el procesador no baje la frecuencia ni entre en estados de ahorro mientras jugás.", "It is about keeping the processor from lowering its frequency or entering power-saving states while you play."],
  ["Se basa en cómo Windows usa la RAM y el disco: caché, archivo de paginación y comportamiento del SSD.", "It is about how Windows uses RAM and disk: cache, page file and SSD behavior."],
  ["Se basa en bajar el ping y la variación (jitter): algoritmos de TCP, ahorro de energía del adaptador y DNS.", "It is about lowering ping and jitter: TCP algorithms, adapter power saving and DNS."],
  ["Se basa en apagar servicios de Windows que casi nadie usa y que consumen CPU o disco en segundo plano.", "It is about turning off Windows services that almost nobody uses and that consume CPU or disk in the background."],
  ["Se basa en cortar telemetría y seguimiento: menos procesos hablando con internet y menos ruido en segundo plano.", "It is about cutting telemetry and tracking: fewer processes talking to the internet and less background noise."],
  ["Se basa en quitar publicidad, widgets y relleno que Windows 11 agrega. Todo es reversible.", "It is about removing ads, widgets and filler that Windows 11 adds. Everything is reversible."],
  ["Se basa en desinstalar apps preinstaladas que no usás. Se quitan solo para tu usuario y se pueden restaurar.", "It is about uninstalling preinstalled apps you do not use. They are removed only for your user and can be restored."],
  ["Se basa en reducir animaciones y efectos visuales para que la interfaz responda más rápido.", "It is about reducing animations and visual effects so the interface responds faster."],
  ["Ajustes de riesgo: algunos desactivan protecciones. Los que afectan a un anti-cheat de kernel piden doble confirmación.", "Risky tweaks: some turn off protections. Those that affect a kernel anti-cheat ask for double confirmation."],
  ["Tu puntaje", "Your score"], ["Muestra cuántas de las optimizaciones recomendadas para tu equipo ya están aplicadas.", "Shows how many of the optimizations recommended for your PC are already applied."],
  ["Perfil recomendado", "Recommended profile"], ["Aplica solo ajustes seguros para tu hardware. Es el botón para empezar si no sabés qué elegir.", "Applies only safe tweaks for your hardware. It is the button to start with if you do not know what to choose."],
  ["Suma ajustes de latencia pensados para shooters y esports. Los que dependen del juego no entran: se prueban aparte.", "Adds latency tweaks designed for shooters and esports. Those that depend on the game are not included: they are tested separately."],
  ["Crea una copia de seguridad de Windows antes de tocar nada. Recomendado la primera vez.", "Creates a Windows backup before touching anything. Recommended the first time."],
  ["Revierte todo lo que aplicó OverdriveTuner y deja Windows como estaba. Siempre pide confirmación.", "Reverts everything OverdriveTuner applied and leaves Windows as it was. It always asks for confirmation."],
  ["Tu equipo", "Your PC"], ["Lo que detectamos de tu PC. Con eso mostramos solo los ajustes que le sirven.", "What we detected on your PC. With that we show only the tweaks that suit it."],
  ["Las secciones", "The sections"], ["Cada categoría agrupa ajustes parecidos. Entrá a cualquiera y tocá «Cómo funciona» para ver su explicación.", "Each category groups similar tweaks. Open any of them and tap «How it works» to see its explanation."],
  ["Qué es esta categoría", "What this category is"], ["Filtra los ajustes de esta sección por nombre o descripción.", "Filters this section's tweaks by name or description."],
  ["Activa de una vez los ajustes seguros y compatibles con tu equipo. Los avanzados nunca se aplican en bloque.", "Turns on all the safe tweaks compatible with your PC at once. Advanced ones are never applied in bulk."],
  ["Deshace solo los ajustes de esta categoría.", "Undoes only this category's tweaks."], ["Un ajuste", "A tweak"],
  ["Nombre, nivel de riesgo (Seguro, Riesgo medio o Avanzado) y qué hace. Si dice «Requiere reiniciar», el cambio se nota después de reiniciar.", "Name, risk level (Safe, Medium risk or Advanced) and what it does. If it says «Requires restart», the change shows up after restarting."],
  ["El interruptor", "The switch"], ["Activa o desactiva ese ajuste. Todo queda guardado en Historial y se puede revertir.", "Turns that tweak on or off. Everything is saved in History and can be reverted."],
  ["Estado del overlay", "Overlay status"], ["Te dice si está abierto, qué juego detectó y qué limita tus FPS en ese momento.", "Tells you whether it is open, which game it detected and what limits your FPS at that moment."],
  ["Abre una guía con un overlay de ejemplo que explica qué significa cada valor.", "Opens a guide with a sample overlay that explains what each value means."],
  ["Así se ve el panel. Se actualiza al cambiar esquina, tamaño y qué mostrar.", "This is how the panel looks. It updates when you change corner, size and what to show."],
  ["Graba cuadro por cuadro. Después se analiza en Benchmark y se detectan los microcortes.", "Records frame by frame. Afterwards it is analyzed in Benchmark and micro-stutters are detected."],
  ["Pegar resultados", "Paste results"], ["Pegá el texto del benchmark de CS2 o de otro programa para guardar sus corridas.", "Paste the text from the CS2 benchmark or another program to save its runs."],
  ["Poné un nombre que describa cómo estaba el PC, por ejemplo «antes» u «optimizado».", "Enter a name that describes how the PC was, for example «before» or «optimized»."],
  ["Marcá las corridas hechas antes de optimizar: sirven de referencia.", "Mark the runs made before optimizing: they serve as a reference."],
  ["Compara estados con estadística: te dice si la diferencia es real o solo ruido entre corridas.", "Compares states with statistics: it tells you whether the difference is real or just noise between runs."],
  ["Corridas del overlay", "Overlay runs"], ["Con «Ver» se dibuja la partida y se analizan los microcortes, con su causa más probable.", "With «View» the match is drawn and the micro-stutters are analyzed, with their most likely cause."],
  ["Cuántos problemas importantes y avisos encontró la revisión de tu equipo.", "How many important problems and warnings the check of your PC found."],
  ["Hz de tu monitor", "Your monitor's Hz"], ["Ponelos para que el diagnóstico compare con la frecuencia que Windows está usando.", "Enter them so the diagnostics can compare with the frequency Windows is using."],
  ["Qué usa el procesador", "What the processor is using"], ["Mide 2 segundos y lista los programas que más consumen. Podés cerrar los que no necesites.", "Measures for 2 seconds and lists the programs that use the most. You can close the ones you do not need."],
  ["Red y ping", "Network and ping"], ["Mide ping, variación y pérdida de paquetes.", "Measures ping, jitter and packet loss."],
  ["Compara Cloudflare, Google, Quad9 y OpenDNS desde tu conexión y te deja aplicar el más rápido.", "Compares Cloudflare, Google, Quad9 and OpenDNS from your connection and lets you apply the fastest."],
  ["Lista los programas con conexiones abiertas. Sirve para encontrar descargas en segundo plano mientras jugás.", "Lists the programs with open connections. Useful for finding background downloads while you play."],
  ["Salud, temperatura, desgaste y espacio libre de tus discos, con consejos.", "Health, temperature, wear and free space of your disks, with tips."],
  ["Qué se puede limpiar", "What can be cleaned"], ["Cada fila muestra cuánto espacio libera. Lo marcado por defecto es seguro.", "Each row shows how much space it frees. What is checked by default is safe."],
  ["Borra solo lo seleccionado. Lo que está en uso se omite sin error.", "Deletes only what is selected. What is in use is skipped without an error."],
  ["Programas de inicio", "Startup programs"], ["Lo que arranca con Windows. Desactivá lo que no necesites para que el PC inicie más rápido y use menos memoria.", "What starts with Windows. Turn off what you do not need so the PC starts faster and uses less memory."],
  ["Revierte todo lo aplicado por OverdriveTuner.", "Reverts everything applied by OverdriveTuner."], ["Línea de tiempo", "Timeline"],
  ["Cada cambio con fecha, resultado y un botón para deshacerlo si sigue aplicado.", "Each change with date, result and a button to undo it if it is still applied."],
  /* Configuración */
  ["Idioma, atajos de teclado, copia de tu configuración y animación de apertura.", "Language, keyboard shortcuts, a copy of your settings and the opening animation."],
  ["Idioma", "Language"], ["Idioma de la interfaz", "Interface language"],
  ["Cambia todos los textos de la aplicación. El panel que se dibuja encima del juego (overlay) mantiene sus textos en español.", "Changes all the application's texts. The panel drawn on top of the game (overlay) keeps its texts in Spanish."],
  ["Español", "Spanish"], ["Idioma cambiado.", "Language changed."],
  ["Atajos globales", "Global shortcuts"], ["Atajos activados", "Shortcuts enabled"],
  ["Funcionan aunque estés dentro del juego. Si otro programa ya usa una combinación, no se registra y te avisamos acá.", "They work even while you are inside the game. If another program already uses a combination, it is not registered and we tell you here."],
  ["Mostrar / ocultar el overlay", "Show / hide the overlay"], ["Empezar / detener grabación", "Start / stop recording"], ["Cambiar de esquina", "Change corner"],
  ["Hacé clic en el recuadro y apretá la combinación nueva (por ejemplo Ctrl+Alt+K). Se necesitan dos modificadores, o uno y una tecla F1–F12.", "Click the box and press the new combination (for example Ctrl+Alt+K). You need two modifiers, or one and an F1–F12 key."],
  ["Restablecer", "Reset"], ["Apretá las teclas…", "Press the keys…"],
  ["Copia de la configuración", "Settings backup"], ["Exportar", "Export"],
  ["Guarda en un archivo tus ajustes, la configuración del overlay y qué optimizaciones tenés aplicadas. Sirve para tener una copia o llevarla a otra PC.", "Saves your settings, the overlay configuration and which optimizations you have applied to a file. Useful for keeping a copy or taking it to another PC."],
  ["Exportar configuración", "Export settings"], ["Configuración exportada.", "Settings exported."],
  ["Elegí un archivo exportado. Primero te mostramos qué se va a hacer. Los ajustes de anti-cheat y los avanzados nunca se aplican solos desde un archivo.", "Choose an exported file. We first show you what will be done. Anti-cheat and advanced tweaks are never applied on their own from a file."],
  ["Importar configuración", "Import settings"], ["El archivo no es un JSON válido.", "The file is not valid JSON."],
  ["Esto es lo que se va a hacer", "This is what will be done"], ["Se guardarán los ajustes del archivo.", "The file's settings will be saved."],
  ["Se guardarán los ajustes y la configuración del overlay del archivo.", "The file's settings and overlay configuration will be saved."],
  ["Optimizaciones a aplicar ({n})", "Optimizations to apply ({1})"], ["Ya aplicadas ({n})", "Already applied ({1})"], ["Se omiten ({n})", "Skipped ({1})"], ["Ninguna.", "None."],
  ["Para aplicar optimizaciones el programa necesita permisos de administrador.", "To apply optimizations the program needs administrator rights."],
  ["Importar configuración", "Import settings"], ["Configuración importada.", "Settings imported."],
  ["Este archivo no es una configuración de OverdriveTuner (o es de una versión incompatible).", "This file is not an OverdriveTuner configuration (or it is from an incompatible version)."],
  ["Animación de apertura", "Opening animation"], ["Mostrar animación al abrir", "Show animation on open"],
  ["Una intro corta con sonido. Se puede saltar con cualquier tecla o clic.", "A short intro with sound. It can be skipped with any key or click."],
  ["Volumen", "Volume"], ["0 la deja muda.", "0 mutes it."], ["Probar", "Test"], ["La reproduce ahora con el volumen elegido.", "Plays it now at the chosen volume."], ["Reproducir animación", "Play animation"],
  ["Apretá cualquier tecla para saltar", "Press any key to skip"],
  ["Ayuda", "Help"], ["Recorrido guiado", "Guided tour"], ["Volvé a ver para qué sirve cada sección y cada botón.", "See again what each section and button is for."], ["Repetir recorrido", "Repeat tour"],
  ["Elegí español o inglés. Cambia todos los textos de la interfaz al instante.", "Choose Spanish or English. It changes all the interface texts instantly."],
  ["Hacé clic en un recuadro y apretá la combinación nueva. Funcionan aunque estés dentro del juego. Si otro programa ya usa una, te avisamos.", "Click a box and press the new combination. They work even inside the game. If another program already uses one, we tell you."],
  ["Exportar e importar", "Export and import"], ["Guarda tu configuración en un archivo y la restaura en otra PC. Antes de importar ves exactamente qué se va a hacer.", "Saves your settings to a file and restores them on another PC. Before importing you see exactly what will be done."],
  ["Activala o desactivala, ajustá el volumen y probala. Se puede saltar con cualquier tecla.", "Turn it on or off, adjust the volume and try it. It can be skipped with any key."],
  ["Segunda confirmación.\nPara desactivar «{}» escribí DESACTIVAR (en inglés: DISABLE):", "Second confirmation.\nTo turn off «{1}» type DISABLE:"],
  ["Cancelado: no se cambió nada.", "Cancelled: nothing was changed."],
  ["{n}% de las optimizaciones recomendadas aplicadas", "{1}% of the recommended optimizations applied"],
].forEach(([a, b]) => T(a, b));

[
  ["requiere que Windows esté en un SSD", "requires Windows to be on an SSD"], ["requiere 16 GB de RAM o más", "requires 16 GB of RAM or more"],
  ["pensado para PC de escritorio", "designed for desktop PCs"], ["pensado para portátiles", "designed for laptops"],
  ["requiere una GPU AMD", "requires an AMD GPU"], ["requiere una GPU NVIDIA", "requires an NVIDIA GPU"], ["requiere una GPU Intel", "requires an Intel GPU"], ["requiere Windows 11", "requires Windows 11"],
  ["Si después de optimizar rindió peor de verdad (el benchmark dice «empeora de verdad»), esta herramienta revierte la mitad de tus ajustes, vos medís, y se repite hasta encontrar cuál es. Tenés {n} ajustes aplicados que se pueden probar. Si el cambio era ruido, lo va a decir.", "If it performed worse after optimizing (the benchmark says «really worse»), this tool reverts half of your tweaks, you measure, and it repeats until it finds which one it is. You have {1} applied tweaks that can be tested. If the change was just noise, it will say so."],
].forEach(([a, b]) => T(a, b));

[
  ["Ventana", "Window"], ["Al tocar la X", "When you click the X"],
  ["La ventana se cierra pero OverdriveTuner sigue activo en la bandeja, junto al reloj (el overlay y las grabaciones no se cortan). Hacé clic en su ícono para volver a abrirlo. Para cerrarlo del todo usá este botón o el clic derecho del ícono.", "The window closes but OverdriveTuner keeps running in the system tray, next to the clock (the overlay and recordings are not interrupted). Click its icon to open it again. To close it completely use this button or right-click the icon."],
  ["Salir por completo", "Quit completely"], ["¿Cerrar OverdriveTuner por completo? También se cierra el overlay.", "Quit OverdriveTuner completely? The overlay is closed too."],
  ["La X y la bandeja", "The X and the tray"], ["Al cerrar la ventana el programa sigue en la bandeja. Desde acá (o con clic derecho en el ícono) lo cerrás por completo.", "When you close the window the program stays in the tray. From here (or right-click on the icon) you quit it completely."],
].forEach(([a, b]) => T(a, b));

/* errores del servidor (atajos, configuración) */
[
  ["Mostrar/ocultar", "Show/hide"], ["Grabar", "Record"], ["Cambiar esquina", "Change corner"],
  ["el atajo necesita al menos un modificador (Ctrl, Alt, Shift o Win) y una tecla", "the shortcut needs at least one modifier (Ctrl, Alt, Shift or Win) and a key"],
  ["modificador no reconocido: {}", "unrecognized modifier: {1}"], ["modificador repetido: {}", "repeated modifier: {1}"], ["tecla no válida: {}", "invalid key: {1}"],
  ["usá al menos dos modificadores (por ejemplo Ctrl+Alt) o una tecla F1-F12, para no pisar el tipeo normal", "use at least two modifiers (for example Ctrl+Alt) or an F1-F12 key, so normal typing is not overridden"],
  ["Ctrl+Alt+Supr está reservado por Windows", "Ctrl+Alt+Del is reserved by Windows"],
  ["El atajo {} está repetido en «{}» y «{}»", "The shortcut {1} is repeated in «{2}» and «{3}»"],
  ["otro programa ya usa este atajo", "another program already uses this shortcut"],
  ["Configuración no válida: {}", "Invalid configuration: {1}"], ["Configuración no válida: el juego del overlay debe ser un programa como cs2.exe", "Invalid configuration: the overlay game must be a program like cs2.exe"],
  ["frecuencia no válida", "invalid refresh rate"], ["idioma no válido", "invalid language"], ["volumen no válido", "invalid volume"], ["petición no válida", "invalid request"],
  ["El juego debe ser un nombre de programa como cs2.exe", "The game must be a program name like cs2.exe"],
].forEach(([a, b]) => T(a, b));

/* diálogos de confirmación */
[
  ["⚠ ANTI-CHEAT", "⚠ ANTI-CHEAT"], ["«{}» desactiva: {}.", "«{1}» turns off: {2}."],
  ["Los anti-cheat de kernel lo exigen: si lo desactivás probablemente no puedas jugar partidas competitivas hasta volver a activarlo.", "Kernel anti-cheats require it: if you turn it off you probably will not be able to play competitive matches until you turn it back on."],
  ["Primera confirmación: ¿querés continuar?", "First confirmation: do you want to continue?"], ["Segunda confirmación.", "Second confirmation."],
  ["Para desactivar «{}» escribí DESACTIVAR (en inglés: DISABLE):", "To turn off «{1}» type DISABLE:"],
  ["Este ajuste es avanzado.", "This tweak is advanced."], ["¿Quieres aplicarlo de todos modos?", "Do you want to apply it anyway?"], ["¿Continuar?", "Continue?"],
  ["Se van a revertir los {n} ajustes aplicados por OverdriveTuner y tu sistema quedará como antes de usarlo.", "The {1} tweaks applied by OverdriveTuner will be reverted and your system will be back to how it was before using it."],
  ["{}: se aplicarán {n} ajustes compatibles con tu equipo.", "{1}: {2} tweaks compatible with your PC will be applied."],
  ["Se creará antes un punto de restauración y podrás revertir cada ajuste.", "A restore point is created first and you can revert each tweak."],
  ["¿Borrar todas las corridas guardadas?", "Delete all saved runs?"], ["¿Borrar el historial? Los ajustes aplicados no se modifican.", "Delete the history? Applied tweaks are not modified."],
  ["Aplicado", "Applied"], ["Revertido", "Reverted"], ["Deshacer", "Undo"], ["Borrar historial", "Clear history"], ["falló", "failed"],
].forEach(([a, b]) => T(a, b));

[
  ["Elegí los modificadores y la tecla. Se necesitan dos modificadores, o uno y una tecla F1–F12.", "Choose the modifiers and the key. You need two modifiers, or one and an F1–F12 key."], ["Tecla", "Key"],
].forEach(([a, b]) => T(a, b));

/* ---------------- rediseño 2.2 ---------------- */
[
  ["Optimizar", "Optimize"], ["Ojo:", "Note:"], ["Confirmar", "Confirm"], ["Continuar", "Continue"], ["Restaurar", "Restore"],
  ["Aplicar", "Apply"], ["Aplicar de todos modos", "Apply anyway"], ["Anti-cheat", "Anti-cheat"], ["Desactivar", "Turn off"],
  ["Segunda confirmación", "Second confirmation"], ["Cerrar programa", "Close program"], ["Cerrar «{}»", "Close «{1}»"],
  ["Perdés lo que no hayas guardado en ese programa.", "You lose anything unsaved in that program."],
  ["Borrar corridas", "Delete runs"], ["Peligro: anti-cheat", "Danger: anti-cheat"], ["Instalar de todos modos", "Install anyway"],
  ["Actualizar todo", "Update all"], ["Windows lo deshizo: desactivalo y volvé a activarlo", "Windows undid it: turn it off and on again"],
  ["{n} de {n} optimizaciones recomendadas aplicadas", "{1} of {2} recommended optimizations applied"],
  ["Para desactivar «{}» escribí DESACTIVAR (en inglés: DISABLE):", "To turn off «{1}» type DISABLE:"],
].forEach(([a, b]) => T(a, b));

/* detección automática del juego */
[
  ["Detectar el juego automáticamente", "Detect the game automatically"],
  ["Reconoce solo qué juego abriste (los conocidos, los que agregues abajo o cualquier programa a pantalla completa) y empieza a medir. No hace falta escribir nada.", "It recognizes which game you opened (known ones, the ones you add below, or any full-screen program) and starts measuring. Nothing to type."],
  ["Mostrar solo mientras juego", "Show only while I play"],
  ["El panel aparece al abrir un juego y se esconde cuando lo cerrás.", "The panel appears when you open a game and hides when you close it."],
  ["Abrir el overlay al iniciar OverdriveTuner", "Open the overlay when OverdriveTuner starts"],
  ["Queda funcionando en segundo plano mientras el programa esté abierto (también en la bandeja).", "It keeps running in the background while the program is open (also in the tray)."],
  ["Solo se usa si apagás la detección automática. Nombre del programa, por ejemplo cs2.exe", "Only used if you turn off automatic detection. Program name, for example cs2.exe"],
  ["esperando un juego", "waiting for a game"],
].forEach(([a, b]) => T(a, b));

[
  ["Frecuencia de actualización", "Refresh rate"], ["Cuántas veces por segundo se actualiza el número de FPS. Más rápido sigue mejor los cambios.", "How many times per second the FPS number updates. Faster follows changes better."],
  ["{n} por segundo", "{1} per second"],
].forEach(([a, b]) => T(a, b));

[
  ["Carpetas de juegos adicionales", "Additional game folders"],
].forEach(([a, b]) => T(a, b));

/* guía «Cómo se miden los FPS» */
[
  ["Cómo se miden los FPS", "How FPS are measured"], ["Quién mide y quién muestra", "Who measures and who displays"],
  ["El juego", "The game"], ["Acá adentro cuenta RTSS", "RTSS counts in here"], ["Memoria compartida", "Shared memory"],
  ["FPS y temperaturas", "FPS and temperatures"], ["solo lee y dibuja", "only reads and draws"], ["solo lee", "only reads"],
  ["OverdriveTuner en sí es externo: no entra al juego, no lo modifica ni le inyecta nada. Solo lee esos números desde afuera y los dibuja en su propia ventana. Lo que entra al juego es RTSS, no OverdriveTuner.", "OverdriveTuner itself is external: it does not enter the game, modify it or inject anything into it. It only reads those numbers from outside and draws them in its own window. What gets into the game is RTSS, not OverdriveTuner."],
  ["Sin RTSS, desde afuera no hay nada que leer.", "Without RTSS there is nothing to read from outside."],
  ["Con RTSS, los números llegan solos.", "With RTSS, the numbers arrive on their own."],
  ["Cómo dejarlo andando", "How to get it running"],
  ["Abrí el juego.", "Open the game."], ["El overlay aparece solo, con los FPS y las temperaturas.", "The overlay appears on its own, with FPS and temperatures."],
  ["Cuándo puede no funcionar", "When it may not work"],
  ["Funciona bien cuando", "Works well when"], ["Puede fallar cuando", "May fail when"],
  ["El juego acepta a RTSS.", "The game accepts RTSS."],
  ["El anti-cheat de un juego no acepta a RTSS. Eso es cosa de RTSS, no de OverdriveTuner: probalo primero en una práctica, no en una partida con ranking.", "A game's anti-cheat does not accept RTSS. That is RTSS's matter, not OverdriveTuner's: try it first in practice, not in a ranked match."],
  ["Anterior", "Previous"], ["Siguiente", "Next"], ["Entendido", "Got it"],
].forEach(([a, b]) => T(a, b));

/* v2.8: tareas programadas, ping a servidores de juegos y chequeos de hardware */
[
  ["Tareas programadas", "Scheduled tasks"], ["Limpieza semanal", "Weekly cleanup"], ["Punto de restauración semanal", "Weekly restore point"],
  ["Borra los archivos temporales, los volcados de errores y las miniaturas (solo lo seguro, igual que la selección por defecto de «Limpieza»). No borra la caché de shaders. Corre sin ventana. Si la PC estaba apagada a esa hora, corre al encenderla.", "Deletes temp files, crash dumps and thumbnails (only the safe items, like the default selection in «Cleanup»). It does not delete the shader cache. It runs without a window. If the PC was off at that time, it runs when you turn it on."],
  ["Crea un punto de restauración de Windows para poder volver atrás si algo sale mal. Requiere la Protección del sistema activada en Windows.", "Creates a Windows restore point so you can go back if something goes wrong. It needs System Protection turned on in Windows."],
  ["Domingo", "Sunday"], ["Lunes", "Monday"], ["Martes", "Tuesday"], ["Miércoles", "Wednesday"], ["Jueves", "Thursday"], ["Viernes", "Friday"], ["Sábado", "Saturday"],
  ["Ejecutar ahora", "Run now"], ["Tarea programada.", "Task scheduled."], ["Tarea quitada.", "Task removed."],
  ["No está programada.", "Not scheduled."],
  ["Programada. Próxima vez: {}", "Scheduled. Next run: {1}"], ["Programada", "Scheduled"],
  [". Todavía no corrió", ". Has not run yet"],
  ["Se está ejecutando. El resultado aparece acá en unos segundos.", "Running now. The result shows up here in a few seconds."],
  ["Elegí el día y la hora y OverdriveTuner hace solo la limpieza de temporales o crea un punto de restauración, sin abrir ventana. Se quitan con el interruptor.", "Pick the day and time and OverdriveTuner cleans temp files or creates a restore point by itself, without opening a window. Turn the switch off to remove them."],
  ["Ping a servidores de juegos", "Ping to game servers"],
  ["Midiendo servidores de juegos (unos 5 segundos)…", "Measuring game servers (about 5 seconds)…"],
  ["Mide la latencia, la variación y la pérdida hacia Steam, FACEIT, EA y las regiones donde suelen estar los servidores. Sirve para saber si tu internet o la distancia es el problema.", "Measures latency, variation and loss to Steam, FACEIT, EA and the regions where servers usually are. It helps tell whether your internet or the distance is the problem."],
  ["Mide el tiempo de conexión hacia cada destino. No es el ping exacto de una partida: los servicios de juegos usan redes de distribución y miden el punto de entrada más cercano; las regiones (centros de datos de AWS) muestran mejor la distancia a los servidores. Una pérdida mayor a 0 % o una variación alta suelen notarse como saltos en el juego.", "Measures connection time to each destination. It is not the exact ping of a match: game services use content networks and measure the nearest entry point; the regions (AWS data centers) show the distance to servers better. Loss above 0% or high variation usually shows up as lag spikes in game."],
  ["Regiones (distancia a los servidores)", "Regions (distance to servers)"], ["Servicios de juegos", "Game services"],
  ["No respondió.", "No response."],
  ["Ping {} ms · mínimo {} · máximo {} · variación {} ms · pérdida {}%", "Ping {1} ms · min {2} · max {3} · variation {4} ms · loss {5}%"],
  ["Sudamérica (São Paulo)", "South America (São Paulo)"], ["EE. UU. Este (Virginia)", "US East (Virginia)"], ["EE. UU. Este (Ohio)", "US East (Ohio)"], ["EE. UU. Oeste (Oregón)", "US West (Oregon)"],
  ["Europa (Fráncfort)", "Europe (Frankfurt)"], ["Europa (Irlanda)", "Europe (Ireland)"], ["Europa (España)", "Europe (Spain)"], ["Asia (Singapur)", "Asia (Singapore)"], ["Asia (Tokio)", "Asia (Tokyo)"],
  ["Abrir página de drivers", "Open drivers page"],
  ["Resizable BAR", "Resizable BAR"], ["Resizable BAR (Smart Access Memory)", "Resizable BAR (Smart Access Memory)"],
  ["Activo: la GPU puede acceder a toda su memoria de video a la vez (ventana de {} MiB).", "On: the GPU can access all its video memory at once ({1} MiB window)."],
  ["Está desactivado (la GPU accede a su memoria de a 256 MiB). Activalo en la BIOS: «Above 4G Decoding» y «Resizable BAR» (en AMD a veces se llama «Smart Access Memory»). Suele dar unos FPS extra en varios juegos. Antes actualizá la BIOS y el driver de la GPU.", "It is off (the GPU accesses its memory 256 MiB at a time). Turn it on in the BIOS: «Above 4G Decoding» and «Resizable BAR» (on AMD it is sometimes called «Smart Access Memory»). It often gives a few extra FPS in several games. Update the BIOS and the GPU driver first."],
  ["Windows no permite saber si está activo en una Radeon. Se ve en el software AMD Adrenalin (Rendimiento > Ajuste) y se activa en la BIOS con «Above 4G Decoding» y «Resizable BAR».", "Windows cannot tell whether it is on for a Radeon. You can see it in the AMD Adrenalin software (Performance > Tuning) and it is turned on in the BIOS with «Above 4G Decoding» and «Resizable BAR»."],
  ["Driver de {}", "Driver for {1}"],
  ["Versión {} del {} (hace {} meses). Un driver viejo puede perder rendimiento y correcciones en juegos nuevos. Bajá el más reciente de la web oficial.", "Version {1} from {2} ({3} months ago). An old driver can lose performance and fixes in new games. Download the latest one from the official website."],
  ["Versión {} del {}: es reciente.", "Version {1} from {2}: it is recent."],
  ["Estado del disco {}", "Disk health: {1}"], ["Saludable.", "Healthy."],
  ["Saludable. Desgaste: {}%.", "Healthy. Wear: {1}%."],
  ["Espacio libre en el disco de Windows", "Free space on the Windows drive"],
  ["Quedan {} GB libres ({}%). Con menos del 10% Windows y los juegos se vuelven más lentos. Usá «Limpieza» para liberar espacio.", "{1} GB free ({2}%). Below 10% Windows and games get slower. Use «Cleanup» to free space."],
  ["Quedan {} GB libres ({}%). Conviene dejar al menos un 20%, sobre todo si es un SSD.", "{1} GB free ({2}%). It is best to keep at least 20% free, especially on an SSD."],
  ["{} GB libres ({}%).", "{1} GB free ({2}%)."],
  ["Guía de ajustes de la BIOS", "BIOS settings guide"],
  ["Esto no se cambia desde Windows: entrá a la BIOS (tecla Supr o F2 al encender) y revisá estos puntos.", "This cannot be changed from Windows: enter the BIOS (Del or F2 key at power-on) and check these points."],
  ["Perfil de memoria {} activado.", "Memory profile {1} turned on."],
  ["«Above 4G Decoding» y «Resizable BAR» activados (con una GPU reciente).", "«Above 4G Decoding» and «Resizable BAR» turned on (with a recent GPU)."],
  ["Si un anti-cheat lo pide: Secure Boot, TPM (fTPM/PTT) y {} / IOMMU activados.", "If an anti-cheat requires it: Secure Boot, TPM (fTPM/PTT) and {1} / IOMMU turned on."],
  ["Modo de arranque UEFI, no Legacy.", "UEFI boot mode, not Legacy."],
  ["Actualizá la BIOS solo si tu placa lo recomienda para tu procesador o memoria, y nunca con el equipo en riesgo de apagarse.", "Update the BIOS only if your board recommends it for your processor or memory, and never when the computer might lose power."],
  ["Una por línea. Todo programa con ventana que esté dentro de estas carpetas cuenta como juego. Los de Steam y Epic se reconocen solos, en cualquier disco; esto es para juegos instalados en otro lado, por ejemplo D:\\Juegos.", "One per line. Any program with a window inside these folders counts as a game. Steam and Epic games are recognized on their own, on any drive; this is for games installed elsewhere, for example D:\\Games."],
].forEach(([a, b]) => T(a, b));

/* actualizaciones */
[
  ["Actualizaciones", "Updates"],
  ["Instalar las actualizaciones automáticamente", "Install updates automatically"],
  ["Cuando sale una versión nueva en GitHub, OverdriveTuner la descarga, comprueba su SHA-256 y la instala sola.", "When a new version is published on GitHub, OverdriveTuner downloads it, checks its SHA-256 and installs it by itself."],
  ["Solo consulta GitHub, sin enviar datos tuyos.", "It only contacts GitHub and sends no data about you."],
  ["Si lo desactivás, el programa se queda en esta versión hasta que apliques el Parche.", "If you turn it off, the program stays on this version until you apply the Patch."],
  ["Versión instalada: {}", "Installed version: {1}"],
  ["Buscar actualizaciones", "Check for updates"], ["Actualizar ahora", "Update now"], ["Reiniciar ahora", "Restart now"], ["Abrir GitHub", "Open GitHub"], ["Después", "Later"],
  ["Buscando actualizaciones…", "Checking for updates…"],
  ["Descargando e instalando la versión {}…", "Downloading and installing version {1}…"],
  ["La versión {} ya está instalada. Reiniciá OverdriveTuner para empezar a usarla.", "Version {1} is already installed. Restart OverdriveTuner to start using it."],
  ["La versión {} ya está instalada.", "Version {1} is already installed."],
  ["Hay una versión nueva: {}.", "A new version is available: {1}."],
  ["Hay una versión nueva: {}. Esta copia no está instalada con el instalador: bajá la nueva desde GitHub.", "A new version is available: {1}. This copy was not installed with the installer: download the new one from GitHub."],
  ["Estás al día: no hay versiones nuevas. Última búsqueda: {}", "You are up to date: no new versions. Last check: {1}"],
  ["Se busca sola cada pocas horas.", "It checks by itself every few hours."],
  ["Todavía no se buscó. Podés buscar a mano cuando quieras.", "Not checked yet. You can check manually whenever you want."],
  ["Qué trae la versión {}", "What version {1} brings"],
  ["Actualizaciones automáticas activadas.", "Automatic updates turned on."],
  ["Actualizaciones automáticas desactivadas: el programa se queda en esta versión hasta que apliques el Parche.", "Automatic updates turned off: the program stays on this version until you apply the Patch."],
  ["OverdriveTuner se va a cerrar y a volver a abrir con la versión nueva. También se reinicia el overlay. ¿Seguimos?", "OverdriveTuner will close and reopen with the new version. The overlay restarts too. Continue?"],
  ["Reiniciar", "Restart"], ["Reiniciando…", "Restarting…"],
  ["Todavía no hay ninguna versión publicada en GitHub.", "There is no version published on GitHub yet."],
  ["Ya se está actualizando desde otra ventana. Probá en un minuto.", "An update is already running from another window. Try again in a minute."],
  ["Esta copia no está instalada con el instalador: descargá la versión nueva desde GitHub.", "This copy was not installed with the installer: download the new version from GitHub."],
  ["GitHub limitó las consultas por ahora. Probá más tarde.", "GitHub is limiting requests for now. Try again later."],
  ["GitHub tardó demasiado en responder. Probá más tarde.", "GitHub took too long to respond. Try again later."],
  ["No se pudo buscar actualizaciones: {}", "Could not check for updates: {1}"],
  ["GitHub respondió con un error ({}). Probá más tarde.", "GitHub returned an error ({1}). Try again later."],
  ["Con la casilla activada, las versiones nuevas de GitHub se descargan, se verifican y se instalan solas. Desactivada, el programa se queda como está hasta que apliques el Parche. «Buscar actualizaciones» consulta a mano.", "With the switch on, new GitHub versions are downloaded, verified and installed by themselves. When off, the program stays as it is until you apply the Patch. «Check for updates» looks manually."],
  ["Versión de demostración: no se descarga nada.", "Demo version: nothing is downloaded."],
].forEach(([a, b]) => T(a, b));

/* liberar RAM */
[
  ["Memoria RAM", "RAM memory"], ["Liberar memoria RAM", "Free RAM"], ["Liberar RAM", "Free RAM"], ["Liberando…", "Freeing…"], ["Archivos temporales", "Temporary files"],
  ["Como «Acelerar» de Microsoft PC Manager: le pide a Windows que devuelva la memoria que los programas abiertos no están usando. Es temporal y los programas pueden tardar un instante en volver a responder. Si hay un juego abierto, no se toca el juego.", "Like «Boost» in Microsoft PC Manager: it asks Windows to take back the memory that open programs are not using. It is temporary and programs may take a moment to respond again. If a game is open, the game is left alone."],
  ["Memoria libre antes: {}", "Free memory before: {1}"], ["Memoria libre después: {}", "Free memory after: {1}"],
  ["Se recortaron {n} programas.", "{1} programs were trimmed."], ["No se tocó {}.", "{1} was left alone."], ["Se vació la memoria en espera.", "The standby memory was emptied."],
  ["Modo demostración: no se tocó la memoria de este equipo.", "Demo mode: this computer's memory was not touched."],
  ["Modo juego", "Game mode"],
  ["DRIVERS", "DRIVERS"],
  ["tu PC", "your PC"],
  ["Abre la página oficial de drivers en tu navegador.", "Opens the official drivers page in your browser."],
  [" · monitor objetivo: {n} Hz (detectado)", " · target monitor: {1} Hz (detected)"],
  [" · monitor objetivo: sin detectar", " · target monitor: not detected"],
  ["Usar el detectado ({n} Hz)", "Use the detected one ({1} Hz)"],
  ["Avisar si la PC se calienta", "Warn if the PC overheats"],
  ["Cuando el procesador o la gráfica pasan el límite, el valor se pone en rojo y el overlay muestra un aviso de texto (sin sonido). La de la gráfica funciona siempre; la del procesador solo si hay una fuente de temperatura (Core Temp o Afterburner abiertos).", "When the processor or graphics card passes the limit, the value turns red and the overlay shows a text warning (no sound). The graphics card one always works; the processor one only if there is a temperature source (Core Temp or Afterburner open)."],
  ["Límites de temperatura", "Temperature limits"],
  ["En grados Celsius, entre 60 y 110.", "In degrees Celsius, between 60 and 110."],
  ["Inicio con Windows", "Start with Windows"],
  ["Iniciar el overlay con Windows", "Start the overlay with Windows"],
  ["Al encender la PC el overlay arranca solo, en segundo plano y sin ventana, y aparece cuando abrís un juego. Usa una tarea programada con permisos de administrador, así que Windows no pide confirmación cada vez. Si lo apagás, el overlay solo se abre cuando abrís OverdriveTuner.", "When the PC starts, the overlay launches by itself, in the background and without a window, and appears when you open a game. It uses a scheduled task with administrator rights, so Windows does not ask for confirmation every time. If you turn it off, the overlay only opens when you open OverdriveTuner."],
  ["El overlay arrancará con Windows.", "The overlay will start with Windows."],
  ["El overlay ya no arranca con Windows.", "The overlay no longer starts with Windows."],
  ["Elegí los programas que querés que se cierren sí o sí cuando se abre un juego, para liberar procesador y memoria. Se cierran todos sus procesos apenas el overlay detecta el juego, y no se vuelven a abrir solos. Nunca se toca el juego ni los procesos de Windows.", "Choose the programs you want closed no matter what when a game opens, to free up processor and memory. All their processes are closed as soon as the overlay detects the game, and they do not reopen by themselves. The game and Windows processes are never touched."],
  ["El Modo juego funciona junto con el overlay, que es el que detecta el juego, y ahora no está abierto.", "Game mode works together with the overlay, which is what detects the game, and it is not open right now."],
  ["Abrir overlay", "Open overlay"],
  ["Se cierran al abrir un juego", "Closed when a game opens"],
  ["Todavía no agregaste ningún programa. Por ejemplo: un navegador, Spotify o un cliente de descargas.", "You have not added any program yet. For example: a browser, Spotify or a download client."],
  ["Quitar", "Remove"],
  ["Agregar", "Add"],
  ["Elegir de los programas abiertos", "Choose from open programs"],
  ["Midiendo qué programas están abiertos (2 segundos)…", "Measuring which programs are open (2 seconds)…"],
  ["{n} MB de memoria", "{1} MB of memory"],
  ["No hay programas para sugerir ahora.", "There are no programs to suggest right now."],
  ["Poné solo el nombre del programa, por ejemplo chrome.exe", "Enter only the program name, for example chrome.exe"],
  ["Ya está en la lista.", "It is already on the list."],
  ["Programa delicado", "Sensitive program"],
  ["Agregar igual", "Add anyway"],
  ["«{}» es parte de Windows o de OverdriveTuner y no se puede cerrar desde acá.", "«{1}» is part of Windows or OverdriveTuner and cannot be closed from here."],
  ["«{}» es parte de Windows o de OverdriveTuner y no se puede cerrar desde el Modo juego.", "«{1}» is part of Windows or OverdriveTuner and cannot be closed from Game mode."],
  ["«{}» suele ser un launcher o parte del anti-cheat de un juego. Si lo cerrás al abrir el juego, el juego puede cerrarse o no dejarte entrar a la partida.\n\n¿Agregarlo igual?", "«{1}» is usually a launcher or part of a game's anti-cheat. If you close it when the game opens, the game may close or not let you into the match.\n\nAdd it anyway?"],
  ["Abrir página oficial", "Open official page"],
  ["Este programa no se instala con winget: abrí su página oficial", "This program is not installed with winget: open its official page"],
  ["Programa desconocido", "Unknown program"],
  ["Instalar el overlay con Windows", "Install the overlay with Windows"],
  ["Temperatura del procesador", "Processor temperature"],
  ["Windows no la entrega en la mayoría de los procesadores. El overlay la lee de Core Temp o de MSI Afterburner, el que tengas abierto. Solo lee: no se mete en el juego. Core Temp es gratis y está en «Instalar programas».", "Windows does not provide it on most processors. The overlay reads it from Core Temp or MSI Afterburner, whichever you have open. It only reads: it does not enter the game. Core Temp is free and is in “Install programs”."],
  ["Al encender la PC el overlay arranca solo, en segundo plano, y aparece cuando abrís un juego. Mientras no hay juego no se ve ninguna ventana: solo un ícono en la bandeja junto al reloj. Usa una tarea programada con permisos de administrador, así que Windows no pide confirmación cada vez. Si lo apagás, el overlay solo se abre cuando abrís OverdriveTuner.", "When the PC starts, the overlay launches by itself, in the background, and appears when you open a game. While there is no game no window is shown: just an icon in the tray next to the clock. It uses a scheduled task with administrator rights, so Windows does not ask for confirmation every time. If you turn it off, the overlay only opens when you open OverdriveTuner."],
  ["Probar ahora", "Test now"],
  ["La tarea de inicio no está registrada.", "The startup task is not registered."],
  ["Tarea registrada. Última vez: {} (se ejecutó bien). Overlay activo ahora: sí.", "Task registered. Last run: {1} (ran fine). Overlay active now: yes."],
  ["Tarea registrada. Última vez: {} (se ejecutó bien). Overlay activo ahora: no.", "Task registered. Last run: {1} (ran fine). Overlay active now: no."],
  ["Tarea registrada. Todavía no se ejecutó (corre al próximo inicio de sesión). Overlay activo ahora: sí.", "Task registered. It has not run yet (it runs at the next sign-in). Overlay active now: yes."],
  ["Tarea registrada. Todavía no se ejecutó (corre al próximo inicio de sesión). Overlay activo ahora: no.", "Task registered. It has not run yet (it runs at the next sign-in). Overlay active now: no."],
  ["El overlay está activo.", "The overlay is active."],
  ["Buscar ajuste o sección (Ctrl+K)", "Search tweak or section (Ctrl+K)"], ["Buscar en todo el programa", "Search the whole program"],
  ["Sección", "Section"], ["No hay resultados.", "No results."], ["Qué hace este ajuste", "What this tweak does"],
  ["Qué hace:", "What it does:"], ["Qué cambia en Windows:", "What it changes in Windows:"], ["Riesgo:", "Risk:"], ["Perfiles:", "Profiles:"],
  ["Probalo:", "Test it:"], ["Reinicio:", "Restart:"], ["Anti-cheat:", "Anti-cheat:"], ["Cómo se deshace:", "How to undo it:"],
  ["1 valor del registro", "1 registry value"], ["{n} valores del registro", "{1} registry values"], ["1 servicio", "1 service"], ["{n} servicios", "{1} services"],
  ["1 script de PowerShell", "1 PowerShell script"], ["{n} scripts de PowerShell", "{1} PowerShell scripts"], ["1 comando de Windows", "1 Windows command"],
  ["{n} comandos de Windows", "{1} Windows commands"], ["el plan de energía", "the power plan"], ["{n} planes de energía", "{1} power plans"],
  ["Seguro: no debería causar problemas.", "Safe: it should not cause problems."],
  ["Riesgo medio: puede no convenirle a todos los equipos; probá tus juegos después.", "Medium risk: it may not suit every PC; test your games afterwards."],
  ["Avanzado: baja la seguridad o la estabilidad; no está en ningún perfil.", "Advanced: it lowers security or stability; it is not in any profile."],
  ["Está en los perfiles Recomendado y Competitivo.", "It is in the Recommended and Competitive profiles."], ["Está en el perfil Recomendado.", "It is in the Recommended profile."],
  ["Está en el perfil Competitivo.", "It is in the Competitive profile."], ["No está en ningún perfil.", "It is not in any profile."],
  ["su efecto depende del juego y del equipo; medilo con Benchmark antes y después.", "its effect depends on the game and the PC; measure it with Benchmark before and after."],
  ["necesita reiniciar la PC para completarse.", "the PC needs a restart to complete it."],
  ["desactiva {}, que exigen los anti-cheat de kernel; pide doble confirmación.", "it turns off {1}, which kernel anti-cheats require; it asks for double confirmation."],
  ["apagá el interruptor y vuelve al valor que tenías antes.", "turn the switch off and it goes back to the value you had before."], ["Ver en Historial", "See it in History"],
  ["Volviendo a escanear el equipo… puede tardar unos 20 segundos.", "Scanning your PC again… it can take about 20 seconds."],
  ["Deshacer", "Undo"], ["Cortar", "Cut"], ["Copiar", "Copy"], ["Pegar", "Paste"], ["Eliminar", "Delete"], ["Supr", "Del"], ["Seleccionar todo", "Select all"],
  ["No se pudo usar el portapapeles.", "Could not use the clipboard."],
  ["Separación del borde", "Distance from the edge"], ["En píxeles, desde la esquina elegida.", "In pixels, from the chosen corner."], ["Horizontal", "Horizontal"], ["Vertical", "Vertical"],
  ["Monitor", "Monitor"], ["En qué pantalla aparece el panel. Si ese monitor se desconecta, vuelve al principal.", "Which screen the panel appears on. If that monitor is disconnected, it goes back to the main one."],
  ["Monitor {n} · {} (principal)", "Monitor {1} · {2} (main)"], ["Monitor {n} · {}", "Monitor {1} · {2}"],
  ["Tiempo de cuadro", "Frame time"], ["Hora", "Time"], ["Juego", "Game"], ["Cuadro", "Frame"], ["oculto", "hidden"], ["Subir", "Up"], ["Bajar", "Down"],
  ["Orden de los datos", "Data order"], ["Arrastrá una fila o usá Subir y Bajar. El número de FPS y el gráfico van siempre arriba.", "Drag a row or use Up and Down. The FPS number and the graph always stay on top."],
  ["Estilos guardados", "Saved styles"], ["Guardá cómo se ve ahora (colores, forma, letra, diseño, tamaño y transparencia) para volver a ese aspecto con un clic.", "Save how it looks now (colors, shape, font, layout, size and transparency) to get back to it with one click."],
  ["Nombre, por ejemplo Competitivo", "Name, for example Competitive"], ["Guardar estilo actual", "Save current style"], ["Todavía no guardaste ninguno.", "You have not saved any yet."],
  ["Usar", "Use"], ["Borrar", "Delete"], ["Una línea", "One line"], ["Tarjeta", "Card"], ["Escribí un nombre para el estilo.", "Type a name for the style."],
  ["Estilo guardado: {}", "Style saved: {1}"], ["Estilo aplicado: {}", "Style applied: {1}"], ["¿Borrar el estilo «{}»?", "Delete the style «{1}»?"], ["Hay 12 estilos guardados: borrá alguno primero.", "There are 12 saved styles: delete one first."],
  ["Tema de colores", "Color theme"], ["Elegí uno listo o cambiá cualquier color abajo: el tema pasa a «Personalizado».", "Pick a ready-made one or change any color below: the theme becomes «Custom»."], ["Personalizado", "Custom"],
  ["Colores", "Colors"], ["Hacé clic en un cuadrado para elegir cualquier color, o escribí el código (por ejemplo #00FF88).", "Click a square to pick any color, or type its code (for example #00FF88)."], ["Volver a los colores originales", "Back to the original colors"],
  ["Fondo", "Background"], ["Número de FPS", "FPS number"], ["Valores", "Values"], ["Etiquetas", "Labels"], ["Gráfico", "Graph"], ["Bien", "Good"], ["Aviso", "Warning"], ["Problema", "Problem"], ["{} (código)", "{1} (code)"],
  ["Clásico", "Classic"], ["Verde neón", "Neon green"], ["Cian", "Cyan"], ["Ámbar", "Amber"], ["Rojo gamer", "Gamer red"], ["Alto contraste", "High contrast"],
  ["FPS de color según el valor", "FPS color by value"], ["El número de FPS usa el color «Problema» por debajo del primer valor, «Aviso» por debajo del segundo y «Bien» desde ahí.", "The FPS number uses the «Problem» color below the first value, «Warning» below the second and «Good» from there on."], ["Problema bajo", "Problem below"], ["Aviso bajo", "Warning below"],
  ["Fondo transparente", "Transparent background"], ["Sin tarjeta: solo los números sobre el juego, con un contorno oscuro para que se lean.", "No card: just the numbers over the game, with a dark outline so they stay readable."],
  ["Esquinas", "Corners"], ["De rectas a muy redondeadas.", "From square to very rounded."], ["Borde", "Border"], ["Sin borde", "No border"], ["Fino", "Thin"], ["Medio", "Medium"], ["Grueso", "Thick"],
  ["Letra", "Font"], ["Consolas tiene todos los números del mismo ancho: el valor no «baila» al cambiar.", "Consolas has all digits the same width: the value does not «jump» when it changes."], ["Normal", "Normal"], ["Seminegrita", "Semibold"], ["Negrita", "Bold"],
  ["Tamaño del número de FPS", "FPS number size"], ["El color tiene que ser un código como #00FF88.", "The color must be a code like #00FF88."],
  ["Antes de empezar", "Before you start"], ["Entiendo y acepto", "I understand and accept"], ["Salir", "Exit"],
  ["OverdriveTuner cambia configuraciones de Windows: registro, servicios, energía, red y programas instalados. Cada cambio guarda una copia y se puede revertir, y antes de aplicar un perfil se crea un punto de restauración, pero ningún optimizador puede garantizar el mismo resultado en todos los equipos.", "OverdriveTuner changes Windows settings: registry, services, power, network and installed programs. Every change keeps a backup and can be reverted, and a restore point is created before applying a profile, but no optimizer can guarantee the same result on every PC."],
  ["Lo usás bajo tu responsabilidad. El programa se entrega «tal cual», sin garantías (licencia MIT): su autor no se hace responsable por fallas, pérdida de datos, problemas con anti-cheat ni otros daños que puedan surgir de su uso.", "You use it at your own risk. The program is provided «as is», without warranty (MIT license): its author is not liable for failures, data loss, anti-cheat problems or any other damage arising from its use."],
  ["Recomendación: aplicá los cambios de a poco, probá tus juegos y revertí lo que no te convenza.", "Tip: apply changes a few at a time, test your games and revert whatever you do not like."],
  ["Lo aplicás bajo tu responsabilidad. ¿Continuar?", "You apply it at your own risk. Continue?"],
  ["Riesgo medio: puede no convenirle a todos los equipos. Probá tus juegos después de aplicarlo; se revierte con el mismo interruptor.", "Medium risk: it may not suit every PC. Test your games after applying it; the same switch reverts it."],
  ["Se aplicarán {n} ajustes de «{}». Cada uno se puede revertir.", "{1} tweaks from «{2}» will be applied. Each one can be reverted."],
  ["Vas a borrar la caché de shaders de DirectX, NVIDIA y AMD.", "You are about to delete the DirectX, NVIDIA and AMD shader cache."],
  ["Los juegos la vuelven a crear solos, pero las primeras partidas después pueden tardar más en cargar y tener tirones (microcortes) mientras se compilan de nuevo. Conviene hacerlo solo si un juego tiene errores gráficos o después de cambiar el driver de la placa de video, y no justo antes de una partida competitiva.", "Games rebuild it on their own, but the first matches afterwards may load slower and stutter while shaders compile again. Only do it if a game has graphical glitches or after changing the graphics driver, and not right before a competitive match."],
  ["Lo hacés bajo tu responsabilidad: los archivos borrados no se pueden recuperar.", "You do it at your own risk: deleted files cannot be recovered."],
  ["Borrar igual", "Delete anyway"], ["Cancelado: no se borró nada.", "Cancelled: nothing was deleted."],
  ["Borrar la caché de shaders necesita la confirmación del usuario.", "Deleting the shader cache needs the user's confirmation."],
  ["El juego está en pantalla completa exclusiva: Windows no deja que ninguna ventana externa (tampoco este overlay) se dibuje encima. Ponelo en «ventana sin bordes» en sus opciones de video. Si aplicaste «Desactivar optimizaciones de pantalla completa», revertilo.", "The game is in exclusive fullscreen: Windows does not let any external window (including this overlay) draw on top. Set it to «borderless window» in its video options. If you applied «Turn off fullscreen optimizations», revert it."],
  ["Se ejecutó la tarea, pero el overlay no aparece activo.", "The task ran, but the overlay does not appear active."],
  ["Tarea registrada. Última vez: {} (terminó con el código {n}). Overlay activo ahora: sí.", "Task registered. Last run: {1} (ended with code {2}). Overlay active now: yes."],
  ["Tarea registrada. Última vez: {} (terminó con el código {n}). Overlay activo ahora: no.", "Task registered. Last run: {1} (ended with code {2}). Overlay active now: no."],
].forEach(([a, b]) => T(a, b));
