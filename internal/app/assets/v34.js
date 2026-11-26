"use strict";
/* 3.3: pruebas de rendimiento, micrófono, todas las opciones de Windows, editor del overlay, actualizar
   programas de a uno y búsqueda ampliada. Los textos van con los dos idiomas juntos: tt(español, inglés). */

const tt = (es, en) => (window.UILANG === "en" ? en : es);
const head34 = (h, p) => `<div class="top"><div><h2>${h}</h2><p>${p}</p></div></div>`;

/* ---------- pruebas de rendimiento ---------- */
let perfTimer = 0, perfQueue = [], perfRoot = "";
const PERF = () => ({
  cpu1: [tt("Procesador, un núcleo", "Processor, single core"), tt("Lo que más pesa en muchos juegos. 1000 puntos = un Ryzen 5 5600G.", "What matters most in many games. 1000 points = a Ryzen 5 5600G."), tt("puntos", "points")],
  cpuall: [tt("Procesador, todos los núcleos", "Processor, all cores"), tt("Todos los hilos a la vez durante 8 segundos.", "All threads at once for 8 seconds."), tt("puntos", "points")],
  ram: [tt("Velocidad de la RAM", "RAM speed"), tt("Lectura, escritura y copia. Los números no son iguales a los de AIDA64: sirven para comparar en este programa, por ejemplo antes y después de activar XMP/DOCP.", "Read, write and copy. The numbers differ from AIDA64: use them to compare within this program, for example before and after turning on XMP/DOCP."), "GB/s"],
  ramlat: [tt("Latencia de la RAM", "RAM latency"), tt("Cuánto tarda un acceso al azar. Menos es mejor.", "How long a random access takes. Lower is better."), "ns"],
  diskr: [tt("Lectura de disco", "Disk read"), tt("Escribe un archivo temporal de hasta 1 GB, lo lee sin la caché de Windows y lo borra.", "Writes a temporary file of up to 1 GB, reads it bypassing the Windows cache and deletes it."), "MB/s"],
  diskw: [tt("Escritura de disco", "Disk write"), tt("Escribe un archivo temporal de hasta 1 GB y lo borra.", "Writes a temporary file of up to 1 GB and deletes it."), "MB/s"],
  stab: [tt("Estabilidad y throttling", "Stability and throttling"), tt("Carga todo el procesador, comprueba cada cuenta y mira si baja la velocidad por temperatura o energía. Es una prueba corta: pasarla no reemplaza horas de prueba si hacés overclock.", "Loads the whole processor, checks every calculation and watches whether it slows down because of heat or power. It is a short test: passing it does not replace hours of testing if you overclock."), tt("puntos", "points")],
});
const pn = v => (v >= 100 ? Math.round(v).toLocaleString() : (+v).toFixed(1));
function perfValue(e) {
  const u = PERF()[e.kind][2], x = e.extra || [];
  let t = `${pn(e.value)} ${u}`;
  if (e.kind === "ram") t += ` · ${tt("escritura", "write")} ${pn(x[0])} · ${tt("copia", "copy")} ${pn(x[1])} GB/s`;
  if (e.kind === "diskr" || e.kind === "diskw") t += ` · 4K ${Math.round(x[0]).toLocaleString()} IOPS (${e.target})`;
  if (e.kind === "stab") t += ` · ${tt("caída", "drop")} ${pn(x[1])}%${x[2] > 0 ? ` · máx. ${Math.round(x[2])} °C` : ""} · ${Math.round(x[3])} s`;
  return t;
}
function stabChart(pts) {
  if (!pts || pts.length < 2) return "";
  const W = 640, H = 120, mr = Math.max(...pts.map(p => p.rate)) || 1, mx = pts[pts.length - 1].t;
  const line = (f, top, cls) => `<polyline class="${cls}" points="${pts.filter(p => f(p) > 0).map(p => `${(p.t / mx * W).toFixed(1)},${(H - f(p) / top * (H - 8)).toFixed(1)}`).join(" ")}"/>`;
  return `<svg class="pf-chart" viewBox="0 0 ${W} ${H}" role="img" aria-label="${tt("Velocidad y temperatura durante la prueba", "Speed and temperature during the test")}">${line(p => p.rate, mr * 1.05, "r")}${line(p => p.temp, 110, "t")}</svg><p class="note">${tt("Blanca: velocidad. Amarilla: temperatura (si se puede leer).", "White: speed. Yellow: temperature (if it can be read).")}</p>`;
}
function stabHtml(r, maxTemp) {
  const L = [];
  if (r.stopped === "heat") L.push(["bad", tt(`Se cortó sola: el procesador llegó a ${maxTemp} °C. Revisá el disipador, la pasta térmica y los ventiladores.`, `It stopped by itself: the processor reached ${maxTemp} °C. Check the cooler, thermal paste and fans.`)]);
  if (r.stopped === "user") L.push(["", tt("La cortaste antes de que termine.", "You stopped it before it finished.")]);
  L.push(r.errors ? ["bad", tt(`${r.errors} cuentas dieron mal: el procesador o la RAM no están estables. Si hiciste overclock o undervolt, aflojalo.`, `${r.errors} calculations were wrong: the processor or RAM is not stable. If you overclocked or undervolted, back off.`)] : ["ok", tt("Todas las cuentas dieron bien.", "Every calculation was correct.")]);
  if (r.throttle === "limit") L.push(["warn", tt(`Throttling: el firmware limitó el procesador hasta el ${Math.round(r.minLimit)}% de su velocidad (por temperatura, energía o plan de energía).`, `Throttling: the firmware limited the processor down to ${Math.round(r.minLimit)}% of its speed (heat, power or power plan).`)]);
  else if (r.throttle === "drop") L.push(["warn", tt(`Throttling probable: terminó un ${pn(r.dropPct)}% más lento que al principio. Puede ser temperatura, el límite de energía u otro programa.`, `Likely throttling: it ended ${pn(r.dropPct)}% slower than it started. It can be heat, the power limit or another program.`)]);
  else if (r.points.length >= 12) L.push(["ok", tt(`Sin throttling: mantuvo la velocidad (caída ${pn(r.dropPct)}%).`, `No throttling: it held its speed (${pn(r.dropPct)}% drop).`)]);
  L.push(r.maxTemp > 0 ? ["", tt(`Temperatura máxima: ${Math.round(r.maxTemp)} °C.`, `Maximum temperature: ${Math.round(r.maxTemp)} °C.`)] : ["", tt("No se pudo leer la temperatura del procesador: instalá Core Temp desde «Instalar programas» para verla y para que la prueba se corte sola si se calienta de más.", "The processor temperature could not be read: install Core Temp from «Install programs» to see it and so the test stops by itself if it overheats.")]);
  return L.map(([c, t]) => `<div class="banner ${c === "bad" ? "bad" : ""}">${c === "ok" ? stIc("ok") : c === "warn" ? stIc("warn") : ""}${esc(t)}</div>`).join("") + stabChart(r.points);
}

async function renderPerf(m) {
  clearTimeout(perfTimer);
  const head = head34(tt("Pruebas de rendimiento", "Performance tests"), tt("Medí el procesador, la RAM y los discos, y compará tus propios resultados. Cerrá los juegos y programas pesados antes. El ranking queda solo en esta PC.", "Measure the processor, RAM and drives, and compare your own results. Close games and heavy programs first. The ranking stays on this PC only."));
  let d;
  try { d = await api("perf"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "perf") return;
  if (!d.running && perfQueue.length && !d.error) { perfStart(perfQueue.shift(), m); return; }
  if (d.error) perfQueue = [];
  const P = PERF(), vols = d.volumes || [];
  if (!perfRoot && vols.length) perfRoot = vols[0].root;
  const best = k => d.list.find(e => e.kind === k && e.rank === 1 && (!e.target || e.target === perfRoot));
  const row = k => {
    const b = best(k), run = d.running === k, la = d.last && d.last.kind === k ? d.last : null;
    return `<div class="row"><div style="min-width:0"><h4>${P[k][0]}</h4><p>${P[k][1]}</p>
      ${run ? `<div class="sp-bar"><i style="width:${Math.round(d.progress * 100)}%"></i></div>` : ""}
      ${la ? `<p><b>${esc(perfValue(la))}</b> ${la.rank === 1 ? `<span class="tag ok">${tt("tu mejor marca", "your best")}</span>` : `<span class="tag">${tt("puesto", "rank")} ${la.rank}</span>`}</p>` : ""}
      ${b && !la ? `<p class="note">${tt("Tu mejor marca", "Your best")}: ${esc(perfValue(b))}</p>` : ""}</div>
      <span class="bar" style="margin:0">${k === "stab" && !d.running ? `<select id="pfSec" aria-label="${tt("Duración", "Duration")}">${[60, 120, 300, 600].map(s => `<option value="${s}">${s / 60} min</option>`).join("")}</select>` : ""}${run ? `<button class="btn" data-stop>${tt("Detener", "Stop")}</button>` : `<button class="btn" data-run="${k}" ${d.running ? "disabled" : ""}>${tt("Probar", "Run")}</button>`}</span></div>`;
  };
  m.innerHTML = head + banners() + (d.error ? `<div class="banner bad">${esc(d.error)}</div>` : "") +
    `<div class="bar"><button class="btn primary" id="pfAll" ${d.running ? "disabled" : ""}>${tt("Correr todas (1 minuto)", "Run all (1 minute)")}</button>
      <label>${tt("Disco", "Drive")} <select id="pfRoot" ${d.running ? "disabled" : ""}>${vols.map(v => `<option value="${esc(v.root)}" ${v.root === perfRoot ? "selected" : ""}>${esc(v.root)} · ${gb(v.free)} ${tt("libres", "free")}</option>`).join("")}</select></label></div>
    <div class="list">${d.kinds.map(row).join("")}</div>
    <div id="pfStab">${d.running === "stab" ? stabChart(d.live) : d.stab ? stabHtml(d.stab, d.maxTemp) : ""}</div>
    ${d.list.length ? `<h3 style="margin-top:22px">${tt("Ranking de tus pruebas", "Your test ranking")}</h3>${d.kinds.filter(k => d.list.some(e => e.kind === k)).map(k => `<h4 style="margin:14px 0 4px">${P[k][0]}</h4><table class="table"><tbody>${d.list.filter(e => e.kind === k).map(e => `<tr><td class="num" style="width:40px">${e.rank}</td><td><b>${esc(perfValue(e))}</b></td><td class="note">${new Date(e.at).toLocaleString()}</td><td style="text-align:right"><button class="btn" data-del="${e.id}">${tt("Borrar", "Delete")}</button></td></tr>`).join("")}</tbody></table>`).join("")}
      <div class="bar" style="margin-top:12px"><button class="btn danger" id="pfClear">${tt("Borrar el ranking", "Clear the ranking")}</button></div>` : ""}`;
  if ($("#pfRoot")) $("#pfRoot").onchange = e => { perfRoot = e.target.value; renderPerf(m); };
  $("#pfAll").onclick = () => { perfQueue = d.kinds.filter(k => k !== "stab"); perfStart(perfQueue.shift(), m); };
  m.querySelectorAll("[data-run]").forEach(b => b.onclick = async () => {
    perfQueue = [];
    const secs = $("#pfSec") ? +$("#pfSec").value : 60;
    if (b.dataset.run === "stab" && !await ask(tt("El procesador va a trabajar al 100%: se va a calentar y la PC puede responder lento. Guardá lo que tengas abierto. La podés detener cuando quieras.", "The processor will run at 100%: it will heat up and the PC may respond slowly. Save your work. You can stop it at any time."), { title: P.stab[0], ok: tt("Empezar", "Start") })) return;
    perfStart(b.dataset.run, m, secs);
  });
  const stop = m.querySelector("[data-stop]"); if (stop) stop.onclick = () => { perfQueue = []; api("perf/run", { cancel: true }).catch(() => { }); };
  m.querySelectorAll("[data-del]").forEach(b => b.onclick = async () => { try { await api("perf/delete", { id: +b.dataset.del }); renderPerf(m); } catch (e) { toast(e.message, 5000); } });
  if ($("#pfClear")) $("#pfClear").onclick = async () => { if (await ask(tt("¿Borrar todos los resultados guardados?", "Delete all saved results?"), { title: tt("Borrar el ranking", "Clear the ranking"), ok: tt("Borrar todo", "Delete all"), danger: true })) { await api("perf/delete", { all: true }); renderPerf(m); } };
  if (d.running) perfTimer = setTimeout(() => { if (view === "perf") renderPerf(m); }, 700);
}
async function perfStart(kind, m, seconds) {
  try { await api("perf/run", { kind, root: perfRoot, seconds: seconds || 60 }); } catch (e) { perfQueue = []; toast(e.message, 6000); }
  renderPerf(m);
}

/* ---------- micrófono ---------- */
let micTimer = 0, micURL = "";
async function renderMic(m) {
  clearTimeout(micTimer);
  if (micURL) { URL.revokeObjectURL(micURL); micURL = ""; }
  const head = head34(tt("Micrófono", "Microphone"), tt("Silenciá el micrófono en todo Windows y probá cómo se escucha. La grabación de prueba queda solo en la memoria y se descarta.", "Mute the microphone across Windows and test how it sounds. The test recording stays in memory only and is discarded."));
  let d, hk;
  try { [d, hk] = await Promise.all([api("mic"), api("hotkeys")]); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "mic") return;
  const key = hk.config.hkMic;
  const V = {
    silent: ["bad", tt("No se escuchó nada. Puede estar silenciado, ser otro micrófono el predeterminado o no tener permiso en Windows.", "Nothing was heard. It may be muted, another microphone may be the default, or it may lack permission in Windows.")],
    low: ["bad", tt("Se escucha muy bajo. Acercate o subí el volumen del micrófono en Windows.", "It is very quiet. Move closer or raise the microphone volume in Windows.")],
    loud: ["bad", tt("Satura: suena distorsionado. Bajá el volumen del micrófono.", "It clips: it sounds distorted. Lower the microphone volume.")],
    noisy: ["", tt("Hay bastante ruido de fondo. Si no hablaste durante toda la grabación, revisá ventiladores o la ganancia del micrófono.", "There is quite a lot of background noise. If you were not talking the whole time, check fans or the microphone gain.")],
    ok: ["ok", tt("Se escucha bien.", "It sounds good.")],
  };
  const r = d.report;
  m.innerHTML = head + banners() + `
  <div class="list">
   <div class="row"><div><h4>${tt("Micrófono silenciado", "Microphone muted")}</h4><p>${d.micError ? `<span class="warn">${esc(d.micError)}</span>` : tt("Silencia el micrófono predeterminado para todos los programas (Discord, juegos, OBS). Queda así hasta que lo vuelvas a activar.", "Mutes the default microphone for every program (Discord, games, OBS). It stays that way until you turn it back on.")}</p></div><button class="sw" id="micSw" role="switch" aria-checked="${d.muted}" aria-label="${tt("Micrófono silenciado", "Microphone muted")}" ${d.micError ? "disabled" : ""}></button></div>
   <div class="row"><div><h4>${tt("Atajo", "Shortcut")}: ${key ? esc(key) : tt("sin asignar", "not set")}</h4><p>${tt("Funciona dentro del juego, mientras el overlay esté abierto. Con atajo, el panel muestra «Mic silenciado».", "Works inside the game while the overlay is open. With a shortcut set, the panel shows «Mic muted».")} ${hk.running ? "" : tt("Ahora el overlay está cerrado.", "The overlay is closed right now.")}</p></div><button class="btn" id="micHk">${tt("Elegir atajo", "Choose shortcut")}</button></div>
   <div class="row"><div style="min-width:0"><h4>${tt("Probar el micrófono (6 segundos)", "Test the microphone (6 seconds)")}</h4><p>${tt("Quedate callado los primeros dos segundos y después hablá normal.", "Stay quiet for the first two seconds and then speak normally.")}</p><div class="sp-bar mic-lv"><i id="micLv" style="width:0"></i></div></div><button class="btn primary" id="micRec">${d.recording ? tt("Detener", "Stop") : tt("Grabar", "Record")}</button></div>
  </div>
  <div id="micOut">${d.error ? `<div class="banner bad">${esc(d.error)}</div>` : ""}${r ? `<div class="banner ${V[r.verdict][0] === "bad" ? "bad" : ""}">${V[r.verdict][0] === "ok" ? stIc("ok") : ""}${V[r.verdict][1]}</div>
    <p class="note">${tt("Voz", "Voice")} ${r.voiceDb} dB · ${tt("ruido de fondo", "background noise")} ${r.noiseDb} dB · ${tt("pico", "peak")} ${r.peakDb} dB</p><div class="bar"><audio id="micPlay" controls></audio><button class="btn" id="micForget">${tt("Descartar", "Discard")}</button></div>` : ""}</div>
  <div class="bar" style="margin-top:14px"><button class="btn" data-set="sound">${tt("Sonido de Windows", "Windows sound settings")}</button><button class="btn" data-set="micprivacy">${tt("Permisos del micrófono", "Microphone permissions")}</button></div>`;
  $("#micSw").onclick = async e => { const b = e.currentTarget, want = b.getAttribute("aria-checked") !== "true"; b.disabled = true; try { const n = await api("mic", { mute: want }); b.setAttribute("aria-checked", String(n.muted)); toast(n.muted ? tt("Micrófono silenciado.", "Microphone muted.") : tt("Micrófono activado.", "Microphone on.")); } catch (x) { toast(x.message, 6000); } b.disabled = false; };
  $("#micHk").onclick = () => goTo("settings", "#stHkH");
  $("#micRec").onclick = async () => { try { await api("mic", d.recording ? { cancel: true } : { record: true }); renderMic(m); } catch (x) { toast(x.message, 6000); } };
  m.querySelectorAll("[data-set]").forEach(b => b.onclick = () => api("opensettings", { page: b.dataset.set }).catch(x => toast(x.message, 5000)));
  if (r) {
    $("#micForget").onclick = async () => { await api("mic", { forget: true }); renderMic(m); };
    try { const q = await fetch("/api/mic/wav", { headers: { "X-Token": window.TOKEN } }); if (q.ok && $("#micPlay")) { micURL = URL.createObjectURL(await q.blob()); $("#micPlay").src = micURL; } } catch (x) { /* sin audio */ }
  }
  const poll = async () => {
    if (view !== "mic") return;
    try {
      const n = await api("mic?level=1");
      if (!n.recording) return renderMic(m);
      $("#micLv").style.width = Math.round(Math.min(1, Math.sqrt(n.level)) * 100) + "%";
    } catch (x) { /* sigue */ }
    micTimer = setTimeout(poll, 120);
  };
  if (d.recording) poll();
}

/* ---------- todas las opciones de Windows ---------- */
let WINOPTS = null, winQ = "";
async function loadWinOpts() { if (!WINOPTS) WINOPTS = await api("winopts"); return WINOPTS; }
async function openWinOpt(id) { try { await api("winopts", { id }); } catch (e) { toast(e.message, 5000); } }
async function renderWinOpts(m) {
  const head = head34(tt("Opciones de Windows", "Windows options"), tt("Todas las opciones del Panel de control en un solo lugar (lo que se conoce como «God Mode»). Abrirlas no cambia nada. Los nombres salen en el idioma de tu Windows.", "Every Control Panel option in one place (known as «God Mode»). Opening them changes nothing. Names appear in your Windows language."));
  m.innerHTML = head + `<p class="note">${tt("Leyendo las opciones de Windows…", "Reading Windows options…")}</p>`;
  let d;
  try { d = await loadWinOpts(); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "winopts") return;
  m.innerHTML = head + `<div class="bar"><input type="search" id="woQ" value="${esc(winQ)}" placeholder="${tt("Buscar una opción…", "Search an option…")}" style="flex:1;min-width:220px"><button class="btn" id="woF">${tt("Abrir como carpeta", "Open as a folder")}</button></div><div id="woL"></div>`;
  const paint = () => {
    const words = fold(winQ).split(/\s+/).filter(Boolean), favs = d.favs || [];
    const hit = d.options.filter(o => words.every(w => fold(o.name + " " + o.group).includes(w)));
    const row = o => `<tr><td><a href="#" class="notr wo" data-id="${esc(o.id)}">${esc(o.name)}</a></td><td style="width:120px;text-align:right"><button class="btn" data-fav="${esc(o.id)}" aria-pressed="${favs.includes(o.id)}">${favs.includes(o.id) ? tt("Quitar", "Remove") : tt("Favorito", "Favorite")}</button></td></tr>`;
    const sec = (n, l, cls = "notr") => l.length ? `<h3 class="${cls}" style="margin-top:18px">${esc(n)}</h3><table class="table"><tbody>${l.map(row).join("")}</tbody></table>` : "";
    const groups = [...new Set(hit.map(o => o.group))];
    $("#woL").innerHTML = (words.length ? "" : sec(tt("Favoritos", "Favorites"), d.options.filter(o => favs.includes(o.id)), "")) + groups.map(g => sec(g || "Windows", hit.filter(o => o.group === g))).join("") || `<p class="note">${tt("No hay resultados.", "No results.")}</p>`;
  };
  paint();
  $("#woQ").oninput = e => { winQ = e.target.value; paint(); };
  $("#woF").onclick = () => api("winopts/folder", {}).catch(e => toast(e.message, 5000));
  $("#woL").onclick = async e => {
    const a = e.target.closest(".wo"), f = e.target.closest("[data-fav]");
    if (a) { e.preventDefault(); openWinOpt(a.dataset.id); }
    if (f) { try { d.favs = (await api("winopts", { id: f.dataset.fav, fav: f.getAttribute("aria-pressed") !== "true" })).favs; paint(); } catch (x) { toast(x.message, 5000); } }
  };
}

/* ---------- programas con versión nueva ---------- */
async function appsUpgrades(el) {
  if (!el) return;
  el.innerHTML = `<p class="note">${tt("Consultando a winget…", "Asking winget…")}</p>`;
  let l;
  try { l = await api("apps/upgrades"); } catch (e) { el.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (!l.length) { el.innerHTML = `<p class="note">${tt("Todo lo que winget conoce está al día.", "Everything winget knows about is up to date.")}</p>`; return; }
  el.innerHTML = `<table class="table"><tbody>${l.map((u, i) => `<tr><td style="width:34px"><input type="checkbox" class="upk" data-i="${i}" aria-label="${esc(u.name)}" ${S.admin ? "" : "disabled"}></td><td><b class="notr">${esc(u.name)}</b><div class="cmd notr">${esc(u.id)}</div></td><td class="num notr">${esc(u.from)} → ${esc(u.to)}</td><td style="text-align:right"><span data-us="${i}"></span></td></tr>`).join("")}</tbody></table>
    <div class="bar" style="margin-top:10px"><button class="btn primary" id="upGo" ${S.admin ? "" : "disabled"}>${tt("Actualizar seleccionados", "Update selected")}</button><button class="btn" id="upAll">${tt("Marcar todos", "Select all")}</button><span class="note">${tt("Cerrá los programas que vayas a actualizar.", "Close the programs you are going to update.")}</span></div>`;
  $("#upAll").onclick = () => { const c = [...el.querySelectorAll(".upk:not(:disabled)")], on = c.some(x => !x.checked); c.forEach(x => x.checked = on); };
  $("#upGo").onclick = async () => {
    const pick = [...el.querySelectorAll(".upk:checked")].map(c => +c.dataset.i);
    if (busyOp || !pick.length) return pick.length ? 0 : toast(tt("Marcá qué actualizar.", "Select what to update."));
    busyOp = true; $("#upGo").disabled = true;
    for (const i of pick) {
      const st = el.querySelector(`[data-us="${i}"]`); if (st) st.textContent = tt("Actualizando…", "Updating…");
      let msg, ok = false;
      try { const r = await api("apps/upgrades", { id: l[i].id }); ok = r.ok; msg = ok ? tt("Listo", "Done") : (r.notes || []).join("; "); } catch (e) { msg = e.message; }
      if (st) { st.textContent = msg; st.style.color = ok ? "var(--ok)" : "var(--bad)"; }
      if (ok) { const c = el.querySelector(`.upk[data-i="${i}"]`); if (c) { c.checked = false; c.disabled = true; } }
    }
    busyOp = false; if ($("#upGo")) $("#upGo").disabled = false;
  };
}

/* ---------- overlay: umbrales, SSD, avisos y editor ---------- */
function ovV34(el, cfg, save, reload) {
  const chk = (k, n, p) => `<div class="row"><div><h4>${n}</h4><p>${p}</p></div><input type="checkbox" data-v34="${k}" aria-label="${n}" ${cfg[k] ? "checked" : ""}></div>`;
  const num = (k, lo, hi) => `<input type="number" data-n34="${k}" min="${lo}" max="${hi}" value="${cfg[k]}" style="width:70px" aria-label="${k}">`;
  const pair = (n, a, b) => `<label>${n} ${num(a, 0, 100)} ${num(b, 0, 100)}</label>`;
  el.innerHTML = `<div class="list" style="margin-top:10px">
   <div class="row"><div><h4>${tt("Editor del panel", "Panel editor")}</h4><p>${tt("Arrastrá el panel a cualquier parte de la pantalla, y sus datos para ordenarlos, sacarlos o agregarlos.", "Drag the panel anywhere on the screen, and its data to reorder, remove or add it.")} ${cfg.freePos ? tt("Ahora usa una posición libre.", "It is using a free position now.") : ""}</p></div><button class="btn primary" id="ovEdit">${tt("Abrir el editor", "Open the editor")}</button></div>
   ${chk("colorLevels", tt("Colores según el uso y la temperatura", "Colors by usage and temperature"), tt("Pinta de «Aviso» y de «Problema» los valores que pasen estos límites. 0 = sin color. La GPU viene en 0 porque al 100% es lo normal jugando.", "Paints values above these limits as «Warning» and «Problem». 0 = no color. GPU starts at 0 because 100% is normal while gaming.") + `<div class="bar" style="margin-top:8px">${pair("CPU %", "cpuWarn", "cpuBad")}${pair("GPU %", "gpuWarn", "gpuBad")}${pair("RAM %", "ramWarn", "ramBad")}<label>${tt("Temperatura", "Temperature")} °C ${num("tempWarn", 40, 100)}</label></div>`)}
   <div class="row"><div><h4>${tt("Límite de temperatura del SSD", "SSD temperature limit")}</h4><p>${tt("Se lee de Windows, sin programas extra. Algunos discos SATA no la informan y se ve «—».", "Read from Windows, no extra programs. Some SATA drives do not report it and show «—».")}</p></div><label>°C ${num("alertSsd", 45, 100)}</label></div>
   ${chk("alertTray", tt("Avisar también con una notificación de Windows", "Also alert with a Windows notification"), tt("Cuando el procesador, la gráfica o el SSD pasan su límite (necesita «Avisar si la PC se calienta»). Como mucho una cada 10 minutos por componente.", "When the processor, graphics card or SSD passes its limit (needs «Alert if the PC gets hot»). At most one every 10 minutes per component."))}
   ${chk("onlyProblem", tt("Mostrar el panel solo si hay un problema", "Show the panel only if there is a problem"), tt("Aparece con FPS por debajo de «Problema bajo», temperatura sobre el límite, RAM llena o throttling, y se esconde solo.", "Appears with FPS below «Problem under», temperature above the limit, full RAM or throttling, and hides by itself."))}
   ${chk("showMic", tt("Mostrar si el micrófono está silenciado", "Show when the microphone is muted"), tt("Solo si elegiste un atajo para silenciarlo.", "Only if you set a shortcut to mute it."))}
  </div>`;
  el.querySelectorAll("[data-v34]").forEach(c => c.onchange = () => { cfg[c.dataset.v34] = c.checked; save(); });
  el.querySelectorAll("[data-n34]").forEach(c => c.onchange = async () => { cfg[c.dataset.n34] = Math.round(+c.value) || 0; await save(); el.querySelectorAll("[data-n34]").forEach(x => x.value = cfg[x.dataset.n34]); });
  $("#ovEdit").onclick = () => ovEditor(cfg, save, reload);
}

const OVL_ROW_OF = { showFps: "FPS", showLow: "1% low", showLow01: "0.1% low", showGraph: "Gráfico de cuadros", showTemps: "Temperaturas" };
async function ovEditor(cfg, save, reload) {
  let mons = [];
  try { mons = (await api("overlay")).monitors || []; } catch (e) { /* 16:9 */ }
  const mon = mons[cfg.monitor] || mons[0] || { width: 1920, height: 1080 };
  const v = document.createElement("div");
  v.className = "veil show";
  v.innerHTML = `<div class="modal dlg ov-ed" role="dialog" aria-modal="true" aria-label="${tt("Editor del panel", "Panel editor")}"><h3>${tt("Editor del panel", "Panel editor")}</h3>
    <div class="ov-ed-in"><div><p class="note">${tt("Tu pantalla. Arrastrá el panel (o usá las flechas). Se ve más grande que en el juego.", "Your screen. Drag the panel (or use the arrow keys). It looks bigger than in the game.")}</p><div class="ov-scr" id="oeScr" style="aspect-ratio:${mon.width}/${mon.height}"><div id="oePan" tabindex="0" role="group" aria-label="${tt("Panel: movelo con las flechas", "Panel: move it with the arrow keys")}"></div></div></div>
    <div><p class="note">${tt("Datos. Arrastrá para ordenar; tocá para sacar o agregar.", "Data. Drag to reorder; click to remove or add.")}</p><div id="oeRows"></div></div></div>
    <div class="end"><button class="btn" id="oeCorner">${tt("Volver a una esquina", "Back to a corner")}</button><button class="btn primary" id="oeOk">${tt("Listo", "Done")}</button></div></div>`;
  document.body.appendChild(v);
  const scr = v.querySelector("#oeScr"), pan = v.querySelector("#oePan"), rows = v.querySelector("#oeRows");
  if (!cfg.freePos) { // se parte de la esquina actual
    cfg.posX = cfg.corner[1] === "r" ? 1 - Math.min(0.3, cfg.offsetX / mon.width) : Math.min(0.3, cfg.offsetX / mon.width);
    cfg.posY = cfg.corner[0] === "b" ? 1 - Math.min(0.3, cfg.offsetY / mon.height) : Math.min(0.3, cfg.offsetY / mon.height);
  }
  const place = () => { pan.style.left = cfg.posX * Math.max(0, scr.clientWidth - pan.offsetWidth) + "px"; pan.style.top = cfg.posY * Math.max(0, scr.clientHeight - pan.offsetHeight) + "px"; };
  const draw = () => { pan.innerHTML = ovPreview(Object.assign({}, cfg, { scale: Math.max(0.62, cfg.scale * 0.62) }), null); place(); };
  const commit = async () => { cfg.freePos = true; await save(); };
  const snap = x => (x < 0.015 ? 0 : x > 0.985 ? 1 : Math.abs(x - 0.5) < 0.015 ? 0.5 : x);
  pan.onpointerdown = e => {
    e.preventDefault(); pan.focus(); pan.setPointerCapture(e.pointerId);
    const x0 = e.clientX - pan.offsetLeft, y0 = e.clientY - pan.offsetTop;
    pan.onpointermove = q => {
      const w = Math.max(1, scr.clientWidth - pan.offsetWidth), h = Math.max(1, scr.clientHeight - pan.offsetHeight);
      cfg.posX = snap(Math.min(1, Math.max(0, (q.clientX - x0) / w))); cfg.posY = snap(Math.min(1, Math.max(0, (q.clientY - y0) / h))); place();
    };
    pan.onpointerup = pan.onpointercancel = () => { pan.onpointermove = pan.onpointerup = pan.onpointercancel = null; commit(); };
  };
  pan.onkeydown = e => {
    const d = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }[e.key]; if (!d) return;
    e.preventDefault();
    cfg.posX = Math.min(1, Math.max(0, cfg.posX + d[0] * 0.02)); cfg.posY = Math.min(1, Math.max(0, cfg.posY + d[1] * 0.02)); place();
  };
  pan.onkeyup = e => { if (e.key.startsWith("Arrow")) commit(); };
  const drawRows = () => {
    const chip = (k, n, on, i) => `<button class="btn oe-chip ${on ? "primary" : ""}" data-sh="${k}" ${i >= 0 ? `draggable="true" data-ri="${i}"` : ""} aria-pressed="${!!on}">${esc(tt(n, trText(n)))}</button>`;
    rows.innerHTML = `<div class="oe-g">${Object.entries(OVL_ROW_OF).map(([k, n]) => chip(k, n, cfg[k], -1)).join("")}</div><div class="oe-g col">${cfg.order.map((k, i) => chip(OVL_ROW_SHOW[k], OVL_ROWS[k], cfg[OVL_ROW_SHOW[k]], i)).join("")}</div>`;
    rows.querySelectorAll("[data-sh]").forEach(b => b.onclick = async () => { cfg[b.dataset.sh] = !cfg[b.dataset.sh]; draw(); drawRows(); await save(); });
    let from = -1;
    rows.querySelectorAll("[data-ri]").forEach(b => {
      b.ondragstart = e => { from = +b.dataset.ri; e.dataTransfer.effectAllowed = "move"; };
      b.ondragover = e => e.preventDefault();
      b.ondrop = async e => { e.preventDefault(); const to = +b.dataset.ri; if (from < 0 || from === to) return; const [k] = cfg.order.splice(from, 1); cfg.order.splice(to, 0, k); draw(); drawRows(); await save(); };
    });
  };
  const close = () => { document.removeEventListener("keydown", esc2, true); v.remove(); reload(); };
  const esc2 = e => { if (e.key === "Escape") { e.stopPropagation(); close(); } };
  document.addEventListener("keydown", esc2, true);
  v.querySelector("#oeOk").onclick = close;
  v.querySelector("#oeCorner").onclick = async () => { cfg.freePos = false; await save(); close(); };
  draw(); drawRows(); pan.focus();
}

/* ---------- Configuración: qué borra la limpieza programada ---------- */
async function schCleanPick(row) {
  if (!row) return;
  let l;
  try { l = await api("clean/plan"); } catch (e) { return; }
  if (!row.isConnected || !l.length) return;
  const box = document.createElement("details");
  box.style.cssText = "margin-top:8px;grid-column:1/-1";
  box.innerHTML = `<summary>${tt("Elegir qué limpiar", "Choose what to clean")} (${l.filter(x => x.on).length})</summary><div class="gp-opts" style="margin-top:8px">${l.map(x => `<label><input type="checkbox" value="${esc(x.id)}" ${x.on ? "checked" : ""}> ${esc(tt(x.name, trText(x.name)))}</label>`).join("")}</div><p class="note">${tt("La caché de shaders no está: borrarla sin querer provoca tirones.", "The shader cache is not here: deleting it by accident causes stutters.")}</p>`;
  row.appendChild(box);
  box.querySelectorAll("input").forEach(c => c.onchange = async () => {
    const ids = [...box.querySelectorAll("input:checked")].map(x => x.value);
    try { await api("schedule", { id: "cleanup", clean: ids }); box.querySelector("summary").textContent = `${tt("Elegir qué limpiar", "Choose what to clean")} (${ids.length})`; }
    catch (e) { c.checked = true; toast(e.message, 5000); }
  });
}

/* ---------- búsqueda global ampliada ---------- */
function goTo(v, sel) {
  view = v; query = ""; render(); $("#main").scrollTop = 0;
  if (!sel) return;
  let n = 0;
  const t = setInterval(() => { // la sección puede tardar en cargar
    const el = view === v && document.querySelector(sel);
    if (el) { clearInterval(t); el.scrollIntoView({ block: "center" }); el.classList.add("flash"); if (el.focus && el.tabIndex >= 0) el.focus({ preventScroll: true }); }
    else if (++n > 40 || view !== v) clearInterval(t);
  }, 150);
}
// [nombre, nombre en inglés, palabras extra, sección, elemento]
const SEARCH34 = [
  ["Limpiar archivos temporales", "Clean temporary files", "basura temp cache junk", "clean", "#doClean"], ["Liberar memoria RAM", "Free up RAM", "memoria acelerar memory", "clean", "#ramBtn"], ["Vaciar la Papelera", "Empty the Recycle Bin", "papelera trash", "clean", "#rbGo"], ["Puntos de restauración", "Restore points", "restaurar restore", "clean", "#rpGo"],
  ["Probar DNS", "Test DNS", "comparar red internet cloudflare google", "fluidez", "#tDns"], ["Probar red y ping", "Test network and ping", "latencia lag internet", "fluidez", "#tNet"], ["Ping a servidores de juegos", "Ping game servers", "lag region", "fluidez", "#tGame"], ["Quién usa la red", "Who is using the network", "consumo descarga", "fluidez", "#tNetMon"], ["Estado de los discos", "Drive health", "smart ssd hdd salud", "fluidez", "#tDisk"], ["Latencia del sistema (DPC)", "System latency (DPC)", "stutter audio", "fluidez", "#tDpc"], ["Reportar un problema", "Report a problem", "bug error issue", "fluidez", "#tIssue"],
  ["Benchmark de procesador", "Processor benchmark", "cpu nucleos core puntos", "perf"], ["Velocidad y latencia de RAM", "RAM speed and latency", "memoria xmp", "perf"], ["Velocidad del disco", "Drive speed", "ssd hdd lectura escritura read write", "perf"], ["Prueba de estabilidad", "Stability test", "stress throttling overclock temperatura", "perf"],
  ["Silenciar micrófono", "Mute microphone", "mic mudo", "mic", "#micSw"], ["Probar micrófono", "Test microphone", "mic grabar", "mic", "#micRec"],
  ["Editor del overlay", "Overlay editor", "mover panel arrastrar posicion", "overlay", "#ovEdit"], ["Colores del overlay", "Overlay colors", "tema umbral", "overlay", "#cTheme"], ["Temperatura del SSD", "SSD temperature", "disco calor", "overlay", "[data-n34=alertSsd]"], ["Avisos de temperatura", "Temperature alerts", "calor caliente notificacion", "overlay", "[data-v34=alertTray]"], ["Grabar una corrida", "Record a run", "fps benchmark", "overlay", "#recLbl"],
  ["Actualizar programas", "Update programs", "winget upgrade version", "install", "#upList"], ["Todas las opciones de Windows", "All Windows options", "god mode panel de control", "winopts"], ["Solucionar problemas", "Troubleshooting", "ayuda tirones lag lento calor cierra problema help", "help"], ["Jugar con el perfil del juego", "Play with the game profile", "abrir lanzar arranque launch", "gamemode", "[data-gp]"], ["Comparar dos partidas", "Compare two runs", "sesiones grabadas diferencia", "bench", ".scmp"],
  ["Idioma", "Language", "english español", "settings", "#stLang"], ["Atajos de teclado", "Keyboard shortcuts", "teclas hotkey combinacion", "settings", "#stHkH"], ["Exportar o importar configuración", "Export or import settings", "copia backup", "settings", "#stCfgH"], ["Animación de apertura", "Opening animation", "intro", "settings", "#stIntroH"], ["Iniciar con Windows", "Start with Windows", "arranque boot", "settings", "#stBootH"], ["Limpieza programada", "Scheduled cleanup", "automatica tareas semanal", "settings", "#stSchH"], ["Actualizar OverdriveTuner", "Update OverdriveTuner", "version nueva", "settings", "#stUpdH"],
];
function searchExtra(words, has) {
  const out = [];
  const secName = id => { const s = [...TOOLS, ...PROGRAM].find(x => x[0] === id); return s ? tt(s[1], trText(s[1])) : ""; };
  for (const [es, en, kw, v, sel] of SEARCH34) if (has(es, en, kw)) out.push({ kind: "act", name: tt(es, en), where: secName(v), run: () => goTo(v, sel) });
  if (APPS && words.join("").length > 2) for (const a of APPS.items) if (has(a.name)) out.push({ kind: "act", name: a.name, where: tt("Instalar programas", "Install programs"), run: () => goTo("install", `tr[data-id="${CSS.escape(a.id)}"]`) });
  if (WINOPTS && words.join("").length > 2) for (const o of WINOPTS.options) if (has(o.name, o.group)) out.push({ kind: "act", name: o.name, where: "Windows", run: () => openWinOpt(o.id) });
  return out;
}
let searchReady = false;
function searchPrepare() { // al usar el buscador se cargan las opciones de Windows y los programas
  if (searchReady) return;
  searchReady = true;
  loadWinOpts().catch(() => { searchReady = false; });
  if (!APPS) api("apps?lang=" + (window.UILANG || "es")).then(r => { if (!APPS) { APPS = r; APPS_LANG = window.UILANG; } }).catch(() => { });
}

/* ---------- arranque optimizado (perfiles por juego) ---------- */
function gpLaunch(el, cfg, save) {
  el.querySelectorAll("[data-glrow]").forEach(row => {
    const i = +row.dataset.glrow, g = cfg.games[i];
    row.innerHTML = `<label>${tt("Cómo abrirlo", "How to open it")}: <input type="text" data-gl value="${esc(g.launch || "")}" placeholder="steam://rungameid/730" style="width:300px" spellcheck="false"></label>
      <button class="btn" data-gf>${tt("Buscar", "Find")}</button><button class="btn primary" data-gp ${g.launch ? "" : "disabled"}>${tt("Jugar", "Play")}</button>`;
    row.querySelector("[data-gl]").onchange = async e => {
      const want = e.target.value.trim();
      g.launch = want; await save();
      if (want && !cfg.games[i].launch) toast(tt("Tiene que ser la ruta completa del .exe o un enlace de Steam, Epic, Ubisoft, EA o Battle.net.", "It has to be the full path of the .exe or a Steam, Epic, Ubisoft, EA or Battle.net link."), 7000);
      gpLaunch(el, cfg, save);
    };
    row.querySelector("[data-gf]").onclick = async e => {
      const b = e.currentTarget; b.disabled = true; b.textContent = tt("Buscando…", "Looking…");
      try {
        const r = await api("gamemode/find", { exe: g.exe });
        if (!r.launch) toast(tt("No lo encontré en las carpetas de Steam, Epic, EA, GOG ni Ubisoft. Pegá la ruta completa del .exe.", "I could not find it in the Steam, Epic, EA, GOG or Ubisoft folders. Paste the full path of the .exe."), 7000);
        else { g.launch = r.launch; await save(); toast(tt("Encontrado.", "Found.")); }
      } catch (x) { toast(x.message, 6000); }
      gpLaunch(el, cfg, save);
    };
    row.querySelector("[data-gp]").onclick = async e => {
      const b = e.currentTarget; b.disabled = true;
      try {
        const r = await api("gamemode/launch", { exe: g.exe });
        toast(r.overlay ? tt("Abriendo el juego. El perfil se aplica apenas arranca y se deshace al cerrarlo.", "Opening the game. The profile applies as soon as it starts and is undone when you close it.") : tt("Abriendo el juego. El overlay no está abierto: sin él el perfil no se aplica.", "Opening the game. The overlay is not open: without it the profile is not applied."), 6000);
      } catch (x) { toast(x.message, 6000); }
      setTimeout(() => { b.disabled = false; }, 4000);
    };
  });
}

/* ---------- comparar dos partidas grabadas ---------- */
function sessCompare(box, list) {
  const table = box.querySelector("table");
  if (list.length < 2 || !table) return;
  table.querySelectorAll("tbody tr").forEach((tr, i) => tr.insertAdjacentHTML("afterbegin", `<td style="width:30px"><input type="checkbox" class="scmp" data-i="${i}" aria-label="${tt("Comparar", "Compare")}"></td>`));
  const bar = document.createElement("div");
  bar.className = "bar"; bar.style.marginTop = "10px";
  bar.innerHTML = `<button class="btn" id="scGo" disabled>${tt("Comparar las dos marcadas", "Compare the two selected")}</button><span class="note">${tt("Marcá dos partidas.", "Select two runs.")}</span>`;
  table.after(bar);
  const out = document.createElement("div"); bar.after(out);
  const picked = () => [...box.querySelectorAll(".scmp:checked")].map(c => list[+c.dataset.i]);
  box.querySelectorAll(".scmp").forEach(c => c.onchange = () => {
    if (picked().length > 2) c.checked = false;
    $("#scGo").disabled = picked().length !== 2;
  });
  $("#scGo").onclick = async () => {
    let a, b;
    try { [a, b] = await Promise.all(picked().sort((x, y) => (x.id < y.id ? -1 : 1)).map(x => api("sessions/get?id=" + x.id))); } catch (e) { return toast(e.message, 5000); }
    const stat = x => { const s = [...(x.frames || [])].sort((p, q) => p - q); return { avg: x.run.avg, low1: x.run.low1, low01: x.run.low01, worst: s.length ? s[s.length - 1] : 0, spm: x.spikes / Math.max(1, x.run.seconds / 60), secs: x.run.seconds }; };
    const A = stat(a), B = stat(b);
    const line = (n, k, unit, lower) => {
      const d = A[k] ? (B[k] - A[k]) / A[k] * 100 : 0, good = lower ? d < -3 : d > 3, bad = lower ? d > 3 : d < -3;
      return `<tr><td>${n}</td><td class="num">${(+A[k]).toFixed(1)} ${unit}</td><td class="num">${(+B[k]).toFixed(1)} ${unit}</td><td class="num" style="color:${good ? "var(--ok)" : bad ? "var(--bad)" : "inherit"}">${d > 0 ? "+" : ""}${d.toFixed(1)}%</td></tr>`;
    };
    const name = x => esc((x.label || tt("sin etiqueta", "no label")) + " · " + String(x.at || "").slice(0, 16).replace("T", " "));
    out.innerHTML = `<table class="table" style="margin-top:12px"><thead><tr><th></th><th>A: ${name(a)}</th><th>B: ${name(b)}</th><th>${tt("Cambio", "Change")}</th></tr></thead><tbody>
      ${line(tt("FPS promedio", "Average FPS"), "avg", "")}${line("1% low", "low1", "")}${line("0.1% low", "low01", "")}${line(tt("Peor cuadro", "Worst frame"), "worst", "ms", true)}${line(tt("Picos por minuto", "Spikes per minute"), "spm", "", true)}${line(tt("Duración", "Length"), "secs", "s")}</tbody></table>
      <p class="note">${tt("Verde = mejoró B, rojo = empeoró. Menos de 3% suele ser ruido: compará partidas del mismo mapa y del mismo tramo.", "Green = B improved, red = it got worse. Under 3% is usually noise: compare runs of the same map and section.")}</p>
      <p class="note" style="margin-top:10px">A</p><canvas id="scA" width="1000" height="110" style="width:100%;border-radius:10px"></canvas><p class="note">B</p><canvas id="scB" width="1000" height="110" style="width:100%;border-radius:10px"></canvas>`;
    drawFrames($("#scA"), a.frames); drawFrames($("#scB"), b.frames);
  };
}

/* ---------- asistente de problemas ---------- */
let helpPick = "";
const HELP = () => [
  ["stutter", tt("Tirones o microcortes", "Stutters or microstutters"), [
    [tt("Grabá una partida con el overlay y mirá los picos.", "Record a game with the overlay and look at the spikes."), tt("Ir al overlay", "Go to the overlay"), () => goTo("overlay", "#recLbl")],
    [tt("Revisá el Diagnóstico: frecuencia del monitor, plan de energía y Game Bar.", "Check Diagnostics: monitor refresh rate, power plan and Game Bar."), tt("Abrir Diagnóstico", "Open Diagnostics"), () => goTo("fluidez")],
    [tt("Medí la latencia del sistema (DPC): un driver puede estar frenando todo.", "Measure system latency (DPC): a driver may be holding everything up."), tt("Medir DPC", "Measure DPC"), () => goTo("fluidez", "#tDpc")],
    [tt("Mirá qué programas usan el procesador en segundo plano y agregalos al Modo juego.", "See which programs use the processor in the background and add them to Game mode."), tt("Abrir Modo juego", "Open Game mode"), () => goTo("gamemode", "#gmPick")],
    [tt("Si empezó después de un ajuste, buscá cuál fue.", "If it started after a tweak, find which one."), tt("Buscar el ajuste culpable", "Find the culprit tweak"), () => goTo("bench", "#bStart")],
  ]],
  ["fps", tt("Pocos FPS", "Low FPS"), [
    [tt("Abrí el overlay en el juego: dice si limita la gráfica o el procesador.", "Open the overlay in the game: it tells you whether the graphics card or the processor is the limit."), tt("Ir al overlay", "Go to the overlay"), () => goTo("overlay")],
    [tt("Probá el procesador y la RAM: con la RAM sin XMP/DOCP el procesador rinde menos.", "Test the processor and RAM: with RAM without XMP/DOCP the processor performs worse."), tt("Pruebas de rendimiento", "Performance tests"), () => goTo("perf")],
    [tt("Hacé la prueba de estabilidad: muestra si el procesador baja la velocidad por temperatura.", "Run the stability test: it shows whether the processor slows down from heat."), tt("Probar estabilidad", "Test stability"), () => goTo("perf")],
    [tt("Aplicá el perfil Recomendado.", "Apply the Recommended profile."), tt("Ir al Inicio", "Go to Home"), () => goTo("home")],
    [tt("Actualizá el driver de video.", "Update the graphics driver."), tt("Instalar programas", "Install programs"), () => goTo("install")],
  ]],
  ["lag", tt("Lag o ping alto", "Lag or high ping"), [
    [tt("Probá la red: ping, variación y pérdida de paquetes.", "Test the network: ping, variation and packet loss."), tt("Probar red", "Test network"), () => goTo("fluidez", "#tNet")],
    [tt("Medí el ping a los servidores de tu juego y elegí la región más cercana.", "Measure the ping to your game's servers and pick the closest region."), tt("Ping a servidores", "Ping servers"), () => goTo("fluidez", "#tGame")],
    [tt("Mirá quién usa la red: una descarga de fondo sube el ping.", "See who is using the network: a background download raises ping."), tt("Quién usa la red", "Who uses the network"), () => goTo("fluidez", "#tNetMon")],
    [tt("Revisá los ajustes de red.", "Check the network tweaks."), tt("Red y ping", "Network and ping"), () => goTo("network")],
  ]],
  ["heat", tt("La PC se calienta", "The PC gets hot"), [
    [tt("Hacé la prueba de estabilidad y mirá la temperatura máxima.", "Run the stability test and look at the maximum temperature."), tt("Probar estabilidad", "Test stability"), () => goTo("perf")],
    [tt("Activá los avisos de temperatura en el overlay (también del SSD).", "Turn on temperature alerts in the overlay (SSD too)."), tt("Ir al overlay", "Go to the overlay"), () => goTo("overlay", "[data-v34=alertTray]")],
    [tt("Limpiá los ventiladores y revisá que el aire circule. Si el procesador pasa de 90 °C, cambiá la pasta térmica.", "Clean the fans and check the airflow. If the processor goes above 90 °C, replace the thermal paste."), "", null],
  ]],
  ["slow", tt("Windows está lento", "Windows is slow"), [
    [tt("Desactivá programas que arrancan con Windows.", "Turn off programs that start with Windows."), tt("Inicio de Windows", "Windows startup"), () => goTo("startup")],
    [tt("Liberá espacio: un disco casi lleno vuelve todo lento.", "Free up space: an almost full drive slows everything down."), tt("Espacio en disco", "Disk space"), () => goTo("space")],
    [tt("Limpiá los temporales y liberá RAM.", "Clean temporary files and free up RAM."), tt("Limpieza", "Cleanup"), () => goTo("clean")],
    [tt("Revisá el estado de los discos.", "Check the drives' health."), tt("Estado de los discos", "Drive health"), () => goTo("fluidez", "#tDisk")],
    [tt("Si hace días que no reiniciás, reiniciá.", "If you have not restarted in days, restart."), "", null],
  ]],
  ["crash", tt("Un juego se cierra o la PC se cuelga", "A game crashes or the PC freezes"), [
    [tt("Hacé la prueba de estabilidad: los errores de cálculo muestran un overclock o una RAM inestable.", "Run the stability test: calculation errors reveal an unstable overclock or RAM."), tt("Probar estabilidad", "Test stability"), () => goTo("perf")],
    [tt("Revisá el Diagnóstico: BIOS y drivers viejos.", "Check Diagnostics: old BIOS and drivers."), tt("Abrir Diagnóstico", "Open Diagnostics"), () => goTo("fluidez")],
    [tt("Si empezó después de un cambio, volvé a una copia de tu configuración.", "If it started after a change, go back to a configuration backup."), tt("Configuración", "Settings"), () => goTo("settings", "#stCfgH")],
    [tt("Si nada sirve, armá un reporte y abrilo en GitHub.", "If nothing helps, put together a report and open it on GitHub."), tt("Reportar un problema", "Report a problem"), () => reportIssue()],
  ]],
  ["mic", tt("No me escuchan o se escucha mal", "People cannot hear me or I sound bad"), [
    [tt("Probá el micrófono: dice si está bajo, satura o tiene ruido.", "Test the microphone: it tells you whether it is quiet, clipping or noisy."), tt("Probar micrófono", "Test microphone"), () => goTo("mic", "#micRec")],
    [tt("Fijate que no esté silenciado.", "Check that it is not muted."), tt("Micrófono", "Microphone"), () => goTo("mic", "#micSw")],
    [tt("Revisá los permisos del micrófono en Windows.", "Check the microphone permissions in Windows."), tt("Abrir permisos", "Open permissions"), () => api("opensettings", { page: "micprivacy" }).catch(e => toast(e.message, 5000))],
  ]],
];
function renderHelp(m) {
  const H = HELP(), cur = H.find(h => h[0] === helpPick);
  m.innerHTML = head34(tt("Solucionar problemas", "Troubleshooting"), tt("Elegí qué te pasa y seguí los pasos en orden. Cada paso te lleva a la herramienta del programa que sirve para eso.", "Choose what is happening and follow the steps in order. Each step takes you to the tool in the program that helps with it.")) +
    `<div class="wz-opts">${H.map(h => `<button class="btn wz ${h[0] === helpPick ? "on" : ""}" data-hp="${h[0]}" aria-pressed="${h[0] === helpPick}">${h[1]}</button>`).join("")}</div>` +
    (cur ? `<h3 style="margin-top:18px">${cur[1]}</h3><div class="list">${cur[2].map((s, i) => `<div class="row"><div><h4>${tt("Paso", "Step")} ${i + 1}</h4><p>${s[0]}</p></div>${s[2] ? `<button class="btn" data-hs="${i}">${s[1]}</button>` : ""}</div>`).join("")}</div>` : `<p class="note" style="margin-top:14px">${tt("Elegí una opción.", "Choose an option.")}</p>`);
  m.querySelectorAll("[data-hp]").forEach(b => b.onclick = () => { helpPick = b.dataset.hp; renderHelp(m); });
  if (cur) m.querySelectorAll("[data-hs]").forEach(b => b.onclick = () => cur[2][+b.dataset.hs][2]());
}
