"use strict";
/* v1.4: tutoriales guiados, historial, microcortes, DNS, red y discos. */

const CAT_INFO = {
  fps: "Esta categoría se basa en sacarle trabajo innecesario al sistema mientras jugás: prioridades, modo juego y planificación de CPU.",
  latency: "Se basa en reducir el retraso entre mover el mouse o apretar una tecla y verlo en pantalla: temporizadores, aceleración y cola de entrada.",
  gpu: "Se basa en cómo Windows y el driver reparten el trabajo de la placa de video: programación por hardware, DWM y modos de presentación.",
  power: "Se basa en que el procesador no baje la frecuencia ni entre en estados de ahorro mientras jugás.",
  memory: "Se basa en cómo Windows usa la RAM y el disco: caché, archivo de paginación y comportamiento del SSD.",
  network: "Se basa en bajar el ping y la variación (jitter): algoritmos de TCP, ahorro de energía del adaptador y DNS.",
  services: "Se basa en apagar servicios de Windows que casi nadie usa y que consumen CPU o disco en segundo plano.",
  privacy: "Se basa en cortar telemetría y seguimiento: menos procesos hablando con internet y menos ruido en segundo plano.",
  debloat: "Se basa en quitar publicidad, widgets y relleno que Windows 11 agrega. Todo es reversible.",
  apps: "Se basa en desinstalar apps preinstaladas que no usás. Se quitan solo para tu usuario y se pueden restaurar.",
  ui: "Se basa en reducir animaciones y efectos visuales para que la interfaz responda más rápido.",
  comfort: "Se basa en pequeños ajustes de uso diario del explorador y de Windows: menos molestias, sin tocar el rendimiento.",
  updates: "Se basa en controlar cuándo y cómo Windows Update instala cosas, para que no te interrumpa mientras jugás.",
  advanced: "Ajustes de riesgo: algunos desactivan protecciones. Los que afectan a un anti-cheat de kernel piden doble confirmación.",
};

/* ---------------- recorrido guiado ---------------- */
let tourState = null;

function tourSteps(id) {
  const T = {
    home: [
      { sel: ".hero .ring", t: "Tu puntaje", x: "Muestra cuántas de las optimizaciones recomendadas para tu equipo ya están aplicadas." },
      { sel: "#pRec", t: "Perfil recomendado", x: "Aplica solo ajustes seguros para tu hardware. Es el botón para empezar si no sabés qué elegir." },
      { sel: "#pComp", t: "Perfil competitivo", x: "Suma ajustes de latencia pensados para shooters y esports. Los que dependen del juego no entran: se prueban aparte." },
      { sel: "#pRp", t: "Punto de restauración", x: "Crea una copia de seguridad de Windows antes de tocar nada. Recomendado la primera vez." },
      { sel: "#pUndo", t: "Restaurar todo", x: "Revierte todo lo que aplicó OverdriveTuner y deja Windows como estaba. Siempre pide confirmación." },
      { sel: ".grid", t: "Tu equipo", x: "Lo que detectamos de tu PC. Con eso mostramos solo los ajustes que le sirven." },
      { sel: "#nav", t: "Las secciones", x: "Cada categoría agrupa ajustes parecidos. Entrá a cualquiera y tocá «Cómo funciona» para ver su explicación." },
    ],
    category: [
      { sel: ".top", t: "Qué es esta categoría", x: () => (CAT_INFO[view] || "") },
      { sel: "#q", t: "Buscar", x: "Filtra los ajustes de esta sección por nombre o descripción." },
      { sel: "#allOn", t: "Aplicar todos los compatibles", x: "Activa de una vez los ajustes seguros y compatibles con tu equipo. Los avanzados nunca se aplican en bloque." },
      { sel: "#allOff", t: "Revertir los de esta sección", x: "Deshace solo los ajustes de esta categoría." },
      { sel: "#list .row", t: "Un ajuste", x: "Nombre, nivel de riesgo (Seguro, Riesgo medio o Avanzado) y qué hace. Si dice «Requiere reiniciar», el cambio se nota después de reiniciar." },
      { sel: "#list .sw", t: "El interruptor", x: "Activa o desactiva ese ajuste. Todo queda guardado en Historial y se puede revertir." },
    ],
    overlay: [
      { sel: "#ovStatus", t: "Estado del overlay", x: "Te dice si está abierto, qué juego detectó y qué limita tus FPS en ese momento." },
      { sel: "#fhBtn", t: "Cómo se miden los FPS", x: "Una guía corta: los FPS salen de RTSS y la temperatura del procesador de Core Temp. El Setup instala los dos y el overlay los abre solo." },
      { sel: "#ovGuideBtn", t: "Entender los números", x: "Abre una guía con un overlay de ejemplo que explica qué significa cada valor." },
      { sel: "#ovPrev", t: "Vista previa", x: "Así se ve el panel. Se actualiza al cambiar esquina, tamaño y qué mostrar." },
      { sel: "#recStart", t: "Grabar una corrida", x: "Graba cuadro por cuadro. Después se analiza en Benchmark y se detectan los microcortes." },
    ],
    bench: [
      { sel: "#bTxt", t: "Pegar resultados", x: "Pegá el texto del benchmark de CS2 o de otro programa para guardar sus corridas." },
      { sel: "#bLbl", t: "Estado", x: "Poné un nombre que describa cómo estaba el PC, por ejemplo «antes» u «optimizado»." },
      { sel: "#bBase", t: "Base", x: "Marcá las corridas hechas antes de optimizar: sirven de referencia." },
      { sel: ".table", t: "Resumen", x: "Compara estados con estadística: te dice si la diferencia es real o solo ruido entre corridas." },
      { sel: "#sesView, .table:last-of-type", t: "Corridas del overlay", x: "Con «Ver» se dibuja la partida y se analizan los microcortes, con su causa más probable." },
    ],
    fluidez: [
      { sel: ".banner", t: "Resumen", x: "Cuántos problemas importantes y avisos encontró la revisión de tu equipo." },
      { sel: "#hz", t: "Hz de tu monitor", x: "Ponelos para que el diagnóstico compare con la frecuencia que Windows está usando." },
      { sel: "#procGo", t: "Qué usa el procesador", x: "Mide 2 segundos y lista los programas que más consumen. Podés cerrar los que no necesites." },
      { sel: "#tNet", t: "Red y ping", x: "Mide ping, variación y pérdida de paquetes." },
      { sel: "#tGame", t: "Ping a servidores de juegos", x: "Mide la latencia, la variación y la pérdida hacia Steam, FACEIT, EA y las regiones donde suelen estar los servidores. Sirve para saber si tu internet o la distancia es el problema." },
      { sel: "#tDns", t: "Probar DNS", x: "Compara Cloudflare, Google, Quad9 y OpenDNS desde tu conexión y te deja aplicar el más rápido." },
      { sel: "#tNetMon", t: "Quién usa la red", x: "Lista los programas con conexiones abiertas. Sirve para encontrar descargas en segundo plano mientras jugás." },
      { sel: "#tDisk", t: "Discos", x: "Salud, temperatura, desgaste y espacio libre de tus discos, con consejos." },
    ],
    clean: [
      { sel: ".table", t: "Qué se puede limpiar", x: "Cada fila muestra cuánto espacio libera. Lo marcado por defecto es seguro." },
      { sel: "#doClean", t: "Limpiar", x: "Borra solo lo seleccionado. Lo que está en uso se omite sin error." },
    ],
    startup: [
      { sel: ".table, .list", t: "Programas de inicio", x: "Lo que arranca con Windows. Desactivá lo que no necesites para que el PC inicie más rápido y use menos memoria." },
    ],
    apps: [
      { sel: ".table", t: "Programas por categoría", x: "Cada fila es un programa. «Instalado» te dice que ya lo tenés. Los que dependen de tu hardware (AMD, NVIDIA, Intel) solo aparecen si tu PC los usa." },
      { sel: ".ck", t: "Marcar lo que querés", x: "Tildá los programas que querés instalar. Hace falta abrir OverdriveTuner como administrador." },
      { sel: "#appsInstall", t: "Instalar seleccionados", x: "Los instala uno por uno con winget, el administrador de paquetes de Windows 11, y muestra el resultado de cada uno." },
      { sel: "#appsUpgrade", t: "Actualizar programas", x: "Pide a winget que actualice todo lo que tengas instalado y se pueda actualizar. Puede tardar varios minutos." },
      { sel: ".table", t: "Si winget no puede instalar un programa", x: "Si winget no está disponible o falla con un programa, esa fila muestra el botón «Abrir web oficial»: te lleva al sitio del fabricante para que lo descargues e instales vos." },
      { sel: ".drv", t: "Drivers de video", x: "Abre la página oficial de NVIDIA, AMD o Intel. Los drivers no se instalan desde acá: los descargás y verificás vos." },
    ],
    gamemode: [
      { sel: "#gmList", t: "Programas que se cierran", x: "La lista de programas que se cierran solos cuando el overlay detecta que abriste un juego, para liberar procesador y memoria. Se cierran todos sus procesos." },
      { sel: "#gmNew", t: "Agregar uno", x: "Escribí el nombre del programa, por ejemplo chrome.exe, y tocá «Agregar». Los programas de Windows y de OverdriveTuner están protegidos y no se pueden cerrar desde acá." },
      { sel: "#gmPick", t: "Elegir de los abiertos", x: "Mide 2 segundos y te sugiere los programas abiertos que más memoria usan. Agregás con un clic." },
      { sel: "#gmList", t: "Launchers y anti-cheat", x: "Si agregás un launcher (Steam, EA, FACEIT…) te avisamos: cerrarlo puede sacarte del juego o impedir que entres a la partida." },
      { sel: "#gmStart", t: "Hace falta el overlay", x: "El Modo juego lo ejecuta el overlay, que es el que detecta el juego. Si no está abierto, este botón lo abre." },
    ],
    settings: [
      { sel: "#stLang", t: "Idioma", x: "Elegí español o inglés. Cambia todos los textos de la interfaz al instante." },
      { sel: "#stHk", t: "Atajos globales", x: "Hacé clic en un recuadro y apretá la combinación nueva. Funcionan aunque estés dentro del juego. Si otro programa ya usa una, te avisamos." },
      { sel: "#stCfg", t: "Exportar e importar", x: "Guarda tu configuración en un archivo y la restaura en otra PC. Antes de importar ves exactamente qué se va a hacer." },
      { sel: "#stBoot", t: "Inicio con Windows", x: "El overlay arranca solo al encender la PC y aparece cuando abrís un juego. Acá ves si quedó activo y podés probarlo sin reiniciar." },
      { sel: "#stSch", t: "Tareas programadas", x: "Elegí el día y la hora y OverdriveTuner hace solo la limpieza de temporales o crea un punto de restauración, sin abrir ventana. Se quitan con el interruptor." },
      { sel: "#stUpd", t: "Actualizaciones", x: "Con la casilla activada, las versiones nuevas de GitHub se descargan, se verifican y se instalan solas. Desactivada, el programa se queda como está hasta que apliques el Parche. «Buscar actualizaciones» consulta a mano." },
      { sel: "#stWin", t: "La X y la bandeja", x: "Al cerrar la ventana el programa sigue en la bandeja. Desde acá (o con clic derecho en el ícono) lo cerrás por completo." },
      { sel: "#stIntro", t: "Animación de apertura", x: "Activala o desactivala, ajustá el volumen y probala. Se puede saltar con cualquier tecla." },
    ],
    history: [
      { sel: "#hsUndo", t: "Restaurar todo", x: "Revierte todo lo aplicado por OverdriveTuner." },
      { sel: "#hsList", t: "Línea de tiempo", x: "Cada cambio con fecha, resultado y un botón para deshacerlo si sigue aplicado." },
    ],
  };
  if (T[id]) return T[id];
  return T.category;
}

function tourIdFor(v) {
  if (v === "home" || v === "apps" || v === "gamemode" || v === "overlay" || v === "bench" || v === "fluidez" || v === "clean" || v === "startup" || v === "history" || v === "settings") return v;
  return "category";
}

function startTour(id, stepsOverride) {
  endTour();
  const raw = stepsOverride || tourSteps(id);
  const steps = raw.filter((s, i) => i === 0 || !s.sel || document.querySelector(s.sel));
  if (!steps.length) return toast("No hay nada que explicar acá todavía.");
  const root = document.createElement("div");
  root.id = "tour";
  root.innerHTML = '<div class="spot"></div><div class="tcard" role="dialog" aria-live="polite"><small></small><h4></h4><p></p><div class="tbtn"><button data-t="skip" class="btn">Saltar</button><span></span><button data-t="prev" class="btn">Anterior</button><button data-t="next" class="btn primary">Siguiente</button></div></div>';
  document.body.appendChild(root);
  tourState = { steps, i: 0, root, closeFn: null };
  root.querySelector(".tbtn").onclick = e => {
    const b = e.target.closest("button"); if (!b) return;
    if (b.dataset.t === "skip") endTour(); else if (b.dataset.t === "prev") goTour(tourState.i - 1); else goTour(tourState.i + 1);
  };
  goTour(0);
}

function goTour(i) {
  const st = tourState; if (!st) return;
  if (i >= st.steps.length) return endTour();
  if (i < 0) i = 0;
  st.i = i;
  const s = st.steps[i], card = st.root.querySelector(".tcard"), spot = st.root.querySelector(".spot");
  card.querySelector("small").textContent = `Paso ${i + 1} de ${st.steps.length}`;
  card.querySelector("h4").textContent = s.t;
  card.querySelector("p").textContent = typeof s.x === "function" ? s.x() : s.x;
  card.querySelector('[data-t="prev"]').disabled = i === 0;
  card.querySelector('[data-t="next"]').textContent = i === st.steps.length - 1 ? "Terminar" : "Siguiente";
  const el = s.sel ? document.querySelector(s.sel) : null;
  if (el) { try { el.scrollIntoView({ block: "center", behavior: "instant" }); } catch (e) { el.scrollIntoView(); } }
  placeTour(el, spot, card);
  card.querySelector('[data-t="next"]').focus();
}

function placeTour(el, spot, card) {
  const vw = window.innerWidth, vh = window.innerHeight, pad = 8;
  if (!el) { spot.style.cssText = "left:50%;top:50%;width:0;height:0"; card.style.left = Math.max(12, (vw - card.offsetWidth) / 2) + "px"; card.style.top = Math.max(12, (vh - card.offsetHeight) / 2) + "px"; return; }
  const r = el.getBoundingClientRect();
  spot.style.left = (r.left - pad) + "px"; spot.style.top = (r.top - pad) + "px";
  spot.style.width = (r.width + pad * 2) + "px"; spot.style.height = (r.height + pad * 2) + "px";
  const cw = card.offsetWidth, ch = card.offsetHeight;
  let top = r.bottom + pad + 14;
  if (top + ch > vh - 12) top = r.top - pad - 14 - ch;
  let left = r.left;
  if (tourState && tourState.side === "right" && r.right + pad + 14 + cw < vw - 12) { top = Math.min(vh - ch - 12, Math.max(12, r.top + r.height / 2 - ch / 2)); left = r.right + pad + 14; }
  if (top < 12) { top = Math.min(vh - ch - 12, Math.max(12, r.top + r.height / 2 - ch / 2)); left = r.right + pad + 14; if (left + cw > vw - 12) left = Math.max(12, r.left - pad - 14 - cw); }
  card.style.left = Math.min(Math.max(12, left), vw - cw - 12) + "px";
  card.style.top = Math.min(Math.max(12, top), vh - ch - 12) + "px";
}

function endTour() {
  if (!tourState) return;
  const f = tourState.closeFn;
  tourState.root.remove(); tourState = null;
  if (f) f();
}

window.addEventListener("keydown", e => {
  if (!tourState) return;
  if (e.key === "Escape") endTour();
  else if (e.key === "ArrowRight" || e.key === "Enter") { e.preventDefault(); goTour(tourState.i + 1); }
  else if (e.key === "ArrowLeft") goTour(tourState.i - 1);
});
window.addEventListener("resize", () => { if (tourState) goTour(tourState.i); });

/* botón fijo «Cómo funciona» y oferta de la primera vez */
function lsGet(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }
function lsSet(k, v) { try { localStorage.setItem(k, v); } catch (e) { /* sin almacenamiento */ } }

(function setupHelp() {
  const b = document.createElement("button");
  b.id = "helpBtn"; b.className = "btn"; b.type = "button";
  b.innerHTML = "<b>?</b> Cómo funciona";
  b.onclick = () => startTour(tourIdFor(view));
  document.body.appendChild(b);
  const tourSeen = () => { api("settings", { tourSeen: true }).catch(() => { }); lsSet("otTourSeen", "1"); };
  if (!(window.BOOT && window.BOOT.tourSeen) && !lsGet("otTourSeen")) {
    const o = document.createElement("div");
    o.id = "tourOffer";
    o.innerHTML = '<b>¿Primera vez?</b><p>Te muestro en 1 minuto para qué sirve cada botón. Después lo encontrás siempre en «Cómo funciona».</p><div class="tbtn"><button class="btn" id="tOffNo">Ahora no</button><span></span><button class="btn primary" id="tOffYes">Empezar recorrido</button></div>';
    setTimeout(() => document.body.appendChild(o), 1800);
    o.addEventListener("click", e => {
      if (e.target.id === "tOffYes") { tourSeen(); o.remove(); view = "home"; render(); setTimeout(() => startTour("home"), 150); }
      if (e.target.id === "tOffNo") { tourSeen(); o.remove(); }
    });
  }
})();

/* ---------------- guía del overlay ---------------- */
function openOverlayGuide() {
  const cfg = { opacity: 1, scale: 1.25, layout: "card", showFps: true, showLow: true, showGraph: true, showCpu: true, showGpu: true, showRam: true, showVram: true, showTemps: true };
  const st = { fps: 238, low1: 126, sample: { cpu: 41, gpu: 70, gpuTemp: 54, cpuTemp: -1, ramUsedMb: 11800, gpuVramMb: 5200 } };
  const g = document.createElement("div");
  g.id = "ovGuide";
  g.innerHTML = `<div class="gbox"><h3>Qué significa cada número</h3><p class="note">Este es un overlay de ejemplo con valores de muestra. Usá «Siguiente» para recorrerlo.</p><div class="ovstage">${ovPreview(cfg, st)}</div><div class="btns" style="margin-top:14px"><button class="btn" id="ogClose">Cerrar</button></div></div>`;
  document.body.appendChild(g);
  const closeG = () => g.remove();
  g.querySelector("#ogClose").onclick = () => { endTour(); closeG(); };
  const steps = [
    { sel: ".ovp .top2 strong", t: "FPS", x: "Cuántos cuadros por segundo dibuja el juego ahora mismo. En un monitor de 240 Hz, lo ideal es estar cerca de 240." },
    { sel: ".ovp .top2 .r", t: "1% low", x: "Promedio de los cuadros más lentos (el 1 % peor). Si está muy por debajo del FPS, hay tirones aunque el promedio se vea alto. No es comparable con el 1% low de otros programas: comparalo solo con otras corridas de esta app." },
    { sel: ".ovp .spk", t: "Gráfico de cuadros", x: "Cada barra es un cuadro y la altura es lo que tardó. Verde es normal, amarillo es un tirón leve y rojo es un pico: eso es lo que sentís como microcorte." },
    { sel: ".ovp > div:nth-child(3)", t: "CPU", x: "Uso total del procesador. Ojo: un juego como CS2 usa pocos hilos, así que el CPU puede marcar 50 % y estar limitado igual por un solo núcleo. La temperatura del CPU sale de Core Temp, que el overlay abre solo en segundo plano (Windows no la entrega sin un driver y OverdriveTuner no instala drivers para no chocar con los anti-cheat de kernel)." },
    { sel: ".ovp > div:nth-child(4)", t: "GPU", x: "Uso del motor 3D de la placa de video y su temperatura. Si pasa de 92 %, el límite lo pone la placa de video. Temperatura: verde hasta 69°, amarillo desde 70° y rojo desde 85°." },
    { sel: ".ovp > div:nth-child(5)", t: "VRAM", x: "Memoria de video en uso. Si se llena, aparecen tirones fuertes: bajá texturas o resolución." },
    { sel: ".ovp > div:nth-child(6)", t: "RAM", x: "Memoria del sistema en uso. Si se acerca a tu total, Windows empieza a usar el disco y se nota." },
    { sel: ".ovp", t: "Qué limita tus FPS", x: "En la pantalla de Overlay, la app combina estos números y te dice si el límite es la placa de video, un hilo del CPU u otra cosa, para saber qué conviene ajustar." },
  ];
  startTour("ovguide", steps);
  tourState.closeFn = closeG; tourState.side = "right"; goTour(0);
}

/* ---------------- historial ---------------- */
async function renderHistory(m) {
  const head = `<div class="top"><div><h2>Historial</h2><p>Todos los cambios aplicados y revertidos, del más reciente al más antiguo. Desde acá podés deshacer lo que siga aplicado.</p></div><div class="btns"><button class="btn danger" id="hsUndo" ${S.admin ? "" : "disabled"}>Restaurar todo</button><button class="btn" id="hsClear">Borrar historial</button></div></div>`;
  let list;
  try { list = await api("history"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "history") return;
  const applied = new Set(S.tweaks.filter(t => t.applied).map(t => t.id));
  m.innerHTML = head + banners() + (list.length ? `<table class="table" id="hsList"><tbody>${list.map((h, i) => `<tr>
    <td class="num">${esc(h.at.slice(0, 16).replace("T", " "))}</td>
    <td><span class="tag">${h.action === "apply" ? "Aplicado" : "Revertido"}</span> <b>${esc((S.tweaks.find(t => t.id === h.id) || {}).name || h.name)}</b>${h.ok ? "" : `<div class="cmd" style="color:var(--bad)">${esc(h.error || "falló")}</div>`}</td>
    <td style="text-align:right">${h.action === "apply" && h.ok && applied.has(h.id) && i === list.findIndex(x => x.id === h.id) ? `<button class="btn" data-u="${esc(h.id)}" ${S.admin ? "" : "disabled"}>Deshacer</button>` : ""}</td></tr>`).join("")}</tbody></table>` : '<div id="hsList"><p class="note">Todavía no hay cambios registrados.</p></div>');
  $("#hsUndo").onclick = undoAll;
  $("#hsClear").onclick = async () => { if (!await ask("¿Borrar el historial? Los ajustes aplicados no se modifican.", { title: "Borrar historial", ok: "Borrar", danger: true })) return; await api("history", {}); renderHistory(m); };
  m.querySelectorAll("[data-u]").forEach(b => b.onclick = async () => {
    b.disabled = true;
    try { const r = await api("revert", { id: b.dataset.u }); toast(r.ok ? "Revertido." : "No se pudo: " + (r.errors || []).join("; "), 5000); await refreshTweaks(); } catch (e) { toast(e.message, 6000); b.disabled = false; }
  });
}

/* ---------------- microcortes ---------------- */
async function showMicrocuts(id) {
  const host = $("#sesView"); if (!host) return;
  const box = document.createElement("div"); box.id = "mcOut"; host.appendChild(box);
  box.innerHTML = '<p class="note">Analizando microcortes…</p>';
  let r;
  try { r = await api("sessions/microcuts?id=" + encodeURIComponent(id)); } catch (e) { box.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
  const cls = r.score >= 85 ? "ok" : r.score >= 60 ? "warn" : "bad";
  box.innerHTML = `<div class="mc">
    <div class="mcscore ${cls}"><b>${r.score}</b><span>de 100</span></div>
    <div><h4>Fluidez: ${r.score >= 85 ? "muy buena" : r.score >= 60 ? "con tirones" : "con muchos microcortes"}</h4>
    <p>${esc(r.cause)}</p>
    <p class="note">${r.count} picos · ${(+r.perMinute).toFixed(1)} por minuto · cuadro típico ${(+r.medianMs).toFixed(1)} ms · peor cuadro ${(+r.worstMs).toFixed(0)} ms${r.periodSec ? ` · se repite cada ~${Math.round(r.periodSec)} s` : ""}.</p>
    ${r.advice.length ? `<ul>${r.advice.map(a => `<li>${esc(a)}</li>`).join("")}</ul>` : ""}
    <p class="note">Es una estimación por patrón: indica la causa más probable, no una certeza. Confirmala con el benchmark antes y después de cada cambio.</p></div></div>`;
}

/* ---------------- DNS, red y discos ---------------- */
function dnsHtml(rs) {
  const rows = rs.map(r => `<div class="row ${r.applied ? "on" : ""}"><div><h4>${esc(r.name)} <span class="tag">${esc(r.ip)}</span> ${r.best ? '<span class="tag" style="color:var(--ok)">más rápido</span>' : ""} ${r.applied ? '<span class="tag">en uso</span>' : ""}</h4>
    <p>${r.median < 0 ? "No respondió desde tu conexión." : `Respuesta típica: <b>${r.median} ms</b>${r.fails ? ` · ${r.fails} consulta(s) fallaron` : ""}.`}</p></div>
    ${r.median < 0 ? "" : r.applied ? `<button class="btn" data-dns-off="${esc(r.tweakId)}">Volver al anterior</button>` : `<button class="btn ${r.best ? "primary" : ""}" data-dns-on="${esc(r.tweakId)}" ${S.admin ? "" : "disabled"}>Usar este DNS</button>`}</div>`).join("");
  return `<p class="note">Mide cuánto tarda cada servidor en responder desde tu conexión. Cambiar de DNS acelera la resolución de nombres (conectar a servidores), no baja el ping dentro de la partida. Se guarda tu configuración anterior.</p><div class="list">${rows}</div>`;
}

async function dnsClick(e) {
  const on = e.target.closest("[data-dns-on]"), off = e.target.closest("[data-dns-off]");
  const b = on || off; if (!b) return;
  b.disabled = true;
  try {
    const r = await api(on ? "apply" : "revert", { id: on ? on.dataset.dnsOn : off.dataset.dnsOff });
    if (!r.ok) throw new Error((r.errors || ["No se pudo cambiar el DNS"]).join("; "));
    toast(on ? "DNS cambiado." : "DNS restaurado.");
    S = await api("state");
    const out = $("#tOut"); if (out) out.innerHTML = dnsHtml(await api("dnsbench"));
  } catch (err) { toast(err.message, 6000); b.disabled = false; }
}

/* Ping a servicios y regiones de juegos: promedio, variación (jitter) y pérdida. */
function gamePingHtml(rs, st) {
  const row = r => `<div class="row"><div><h4>${esc(r.name)} ${st(r.status)}</h4><p>${r.status === "error" ? "No respondió." : `Ping ${r.avg} ms · mínimo ${r.min} · máximo ${r.max} · variación ${r.jitter} ms · pérdida ${r.lossPct}%`}</p></div></div>`;
  const g = k => rs.filter(r => r.group === k);
  return `<p class="note">Mide el tiempo de conexión hacia cada destino. No es el ping exacto de una partida: los servicios de juegos usan redes de distribución y miden el punto de entrada más cercano; las regiones (centros de datos de AWS) muestran mejor la distancia a los servidores. Una pérdida mayor a 0 % o una variación alta suelen notarse como saltos en el juego.</p>
  <h3>Regiones (distancia a los servidores)</h3><div class="list">${g("regions").map(row).join("")}</div>
  <h3 style="margin-top:18px">Servicios de juegos</h3><div class="list">${g("games").map(row).join("")}</div>`;
}

function netMonHtml(ps) {
  if (!ps.length) return '<p class="note">No se encontraron programas con conexiones abiertas.</p>';
  return `<p class="note">Programas con conexiones abiertas. La columna de actividad es una estimación (incluye otras entradas y salidas del programa): sirve para encontrar quién usa tu red mientras jugás.</p>
  <table class="table"><thead><tr><th>Programa</th><th>TCP</th><th>UDP</th><th>Actividad</th><th>Se conecta a</th></tr></thead><tbody>${ps.map(p => `<tr><td><b>${esc(p.name)}</b></td><td class="num">${p.tcp}</td><td class="num">${p.udp}</td><td class="num">${(+p.kbs).toFixed(1)} KB/s</td><td>${p.remote.map(esc).join(", ") || "—"}</td></tr>`).join("")}</tbody></table>`;
}

function diskRow(d, st) {
  const bits = [d.media + (d.bus ? " " + d.bus : ""), d.sizeGB + " GB"];
  if (d.temp > 0) bits.push(d.temp + " °C");
  if (d.wear >= 0) bits.push("desgaste " + d.wear + " %");
  if (d.hours >= 0) bits.push(d.hours.toLocaleString("es") + " h encendido");
  if (d.trim) bits.push("TRIM " + (d.trim === "on" ? "activo" : "desactivado"));
  const vols = d.volumes.map(v => { const used = v.sizeGB ? Math.round((1 - v.freeGB / v.sizeGB) * 100) : 0; return `<div class="vol"><span>${esc(v.letter)}:</span><i><b style="width:${used}%" class="${used >= 90 ? "bd" : ""}"></b></i><em>${v.freeGB} GB libres de ${v.sizeGB} GB</em></div>`; }).join("");
  return `<div class="row"><div><h4>${esc(d.name)} ${st(d.status)}</h4><p>${esc(bits.join(" · "))}</p><p>${esc(d.detail)}</p>${vols}${d.tips.map(t => `<p class="note">${esc(t)}</p>`).join("")}</div></div>`;
}
