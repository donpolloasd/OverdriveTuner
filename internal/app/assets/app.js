"use strict";
const $ = (s, el = document) => el.querySelector(s);
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
let S = null;            // estado del servidor
let view = "home";       // sección actual
let query = "";          // filtro de búsqueda
let busyOp = false;      // hay una operación en curso

async function api(path, body) {
  const opt = { headers: { "X-Token": window.TOKEN } };
  if (body !== undefined) { opt.method = "POST"; opt.headers["Content-Type"] = "application/json"; opt.body = JSON.stringify(body); }
  if (path === "state" && window.UILANG) path += "?lang=" + window.UILANG;
  const r = await fetch("/api/" + path, opt);
  let data = null;
  try { data = await r.json(); } catch (e) { /* respuesta vacía */ }
  if (!r.ok) throw new Error((data && data.error) || ("Error " + r.status));
  return data;
}

function toast(msg, ms = 3500) {
  const t = document.createElement("div");
  t.className = "toast"; t.textContent = msg; t.setAttribute("role", "status");
  document.body.appendChild(t);
  setTimeout(() => t.remove(), ms);
}

/* ---------- modal de progreso ---------- */
const modal = {
  open(title) {
    $("#mtitle").textContent = title; $("#mlog").innerHTML = ""; $("#mbar").style.width = "0";
    $("#mclose").disabled = true; $("#veil").classList.add("show");
  },
  log(text, cls = "") { const d = document.createElement("div"); if (cls) d.className = cls; d.textContent = text; const l = $("#mlog"); l.appendChild(d); l.scrollTop = l.scrollHeight; },
  progress(p) { $("#mbar").style.width = Math.round(p * 100) + "%"; },
  done() { $("#mclose").disabled = false; $("#mclose").focus(); },
};
$("#mclose").onclick = () => { $("#veil").classList.remove("show"); render(); };


/* ---------- diálogos propios (reemplazan a confirm/prompt del navegador) ---------- */
function ask(msg, o = {}) {
  return new Promise(res => {
    const v = document.createElement("div");
    v.className = "veil show"; v.style.zIndex = 90;
    v.innerHTML = `<div class="modal dlg ${o.danger ? "danger" : ""}" role="alertdialog" aria-modal="true"><h3></h3><div class="msg"></div>${o.word ? '<input type="text" autocomplete="off" spellcheck="false">' : ""}<div class="end"><button class="btn" data-k="0"></button><button class="btn ${o.danger ? "danger" : "primary"}" data-k="1"></button></div></div>`;
    const set = (sel, t) => { v.querySelector(sel).textContent = t; };
    set("h3", o.title || "Confirmar"); set(".msg", trDialog(msg));
    set('[data-k="0"]', o.cancel || "Cancelar"); set('[data-k="1"]', o.ok || "Continuar");
    const prev = document.activeElement, inp = v.querySelector("input");
    const close = r => { document.removeEventListener("keydown", key, true); v.remove(); if (prev && prev.focus) try { prev.focus(); } catch (e) { /* sin foco */ } res(r); };
    const accept = () => { if (o.word) { const w = inp.value.trim().toUpperCase(); return close(o.word.includes(w) && w ? w : null); } close(true); };
    const key = e => { if (e.key === "Escape") { e.stopPropagation(); close(o.word ? null : false); } else if (e.key === "Enter" && document.activeElement !== v.querySelector('[data-k="0"]')) { e.preventDefault(); accept(); } };
    document.addEventListener("keydown", key, true);
    v.querySelector('[data-k="0"]').onclick = () => close(o.word ? null : false);
    v.querySelector('[data-k="1"]').onclick = accept;
    v.addEventListener("mousedown", e => { if (e.target === v) close(o.word ? null : false); });
    document.body.appendChild(v);
    (inp || v.querySelector('[data-k="0"]')).focus();
  });
}

/* ---------- carga de estado ---------- */
function hideBoot() {
  const go = () => { const b = $("#boot"); if (b) { b.classList.add("gone"); setTimeout(() => b.remove(), 400); } };
  if (window.introActive) { window.introReady = go; } else go();
}

async function load() {
  S = await api("state");
  const want = effectiveLang(S.settings && S.settings.lang);
  if (want !== (window.UILANG || "es")) { applyLang(want); if (want !== S.lang) S = await api("state"); }
  $("#ver").textContent = "v" + S.version;
  render();
  hideBoot();
  // El escaneo del equipo sigue en segundo plano: se completa solo cuando termina.
  while (S.scanning) {
    await new Promise(r => setTimeout(r, 700));
    try { S = await api("state"); } catch (e) { continue; }
    render();
  }
}

function tweaksOf(cat) { return S.tweaks.filter(t => t.cat === cat); }
function activeCount(cat) { const l = tweaksOf(cat).filter(t => t.suitable); return [l.filter(t => t.active).length, l.length]; }

/* ---------- navegación ---------- */
let drvOpen = false;
function renderNav() {
  const btn = (id, name, cnt) => `<button data-v="${id}" class="${view === id ? "on" : ""}" ${view === id ? 'aria-current="page"' : ""}><span>${esc(name)}</span>${cnt || ""}</button>`;
  let html = btn("home", "Inicio");
  html += '<div class="grp">Optimizar</div>';
  for (const c of S.categories) html += btn(c.id, c.name, (([a, n]) => `<span class="cnt">${a}/${n}</span>`)(activeCount(c.id)));
  html += '<div class="grp">Herramientas</div>';
  for (const [id, name] of [["apps", "Instalar programas"], ["overlay", "Overlay en juego"], ["gamemode", "Modo juego"], ["fluidez", "Diagnóstico"], ["bench", "Benchmark"], ["clean", "Limpieza"], ["startup", "Inicio de Windows"]]) html += btn(id, name);
  html += '<div class="grp">Programa</div>';
  html += btn("history", "Historial") + btn("settings", "Configuración");
  $("#nav").innerHTML = html;
  $("#nav").querySelectorAll("button").forEach(b => b.onclick = () => { view = b.dataset.v; query = ""; render(); $("#main").scrollTop = 0; });
  const hwv = S.hardware || {}, mine = (hwv.gpus || []).map(g => g.vendor).concat(hwv.cpuVendor || "");
  $("#foot").innerHTML = `<div class="drv-box"><button class="drv-btn" id="drvBtn" aria-expanded="${drvOpen}">DRIVERS</button>
    <div class="drv-list" id="drvList" ${drvOpen ? "" : "hidden"}>${[["nvidia", "NVIDIA"], ["amd", "AMD"], ["intel", "Intel"]].map(([v, n]) => `<button class="drv-v" data-v="${v}"><span>${n}</span>${mine.includes(v) ? "<small>tu PC</small>" : ""}</button>`).join("")}
    <p class="note">Abre la página oficial de drivers en tu navegador.</p></div></div>` +
    (S.admin ? '<span class="pill ok">Administrador</span>' : '<span class="pill bad">Sin permisos de administrador</span>');
  $("#drvBtn").onclick = () => { drvOpen = !drvOpen; $("#drvList").hidden = !drvOpen; $("#drvBtn").setAttribute("aria-expanded", drvOpen); };
  $("#foot").querySelectorAll(".drv-v").forEach(b => b.onclick = async () => {
    try { await api("apps/drivers", { vendor: b.dataset.v }); } catch (e) { toast("No se pudo abrir: " + e.message, 6000); }
  });
}

function render() {
  if (!S) return;
  renderNav();
  const m = $("#main");
  if (view === "home") return renderHome(m);
  if (view === "apps") return renderApps(m);
  if (view === "overlay") return renderOverlay(m);
  if (view === "gamemode") return renderGameMode(m);
  if (view === "fluidez") return renderFluidez(m);
  if (view === "bench") return renderBench(m);
  if (view === "clean") return renderClean(m);
  if (view === "startup") return renderStartup(m);
  if (view === "history") return renderHistory(m);
  if (view === "settings") return renderSettings(m);
  renderCategory(m, view);
}

function banners() {
  let h = "";
  if (!S.admin) h += '<div class="banner bad">El programa no tiene permisos de administrador, así que no puede aplicar cambios. Ciérralo y ábrelo con clic derecho → «Ejecutar como administrador».</div>';
  if (S.platform !== "windows") h += '<div class="banner">Modo de demostración: estás viendo la interfaz fuera de Windows. Los cambios son simulados y no tocan tu sistema.</div>';
  return h;
}

/* ---------- inicio ---------- */
function score() {
  const rec = S.tweaks.filter(t => t.rec && t.suitable && t.cat !== "apps" && t.risk < 2);
  if (!rec.length) return [0, 0, 0];
  const on = rec.filter(t => t.active).length;
  return [Math.round(on * 100 / rec.length), on, rec.length];
}

function hwCard(label, main, sub) {
  return `<div class="hw"><small>${esc(label)}</small><b>${esc(main)}</b>${sub ? `<span>${esc(sub)}</span>` : ""}</div>`;
}

function renderHome(m) {
  const h = S.hardware, [pct, on, total] = score();
  let cards = S.scanning ? '<div class="skel"></div>'.repeat(6) : hwCard("Equipo", h.computer || "—", h.board);
  if (!S.scanning) cards += hwCard("Procesador", h.cpu || "—", h.cores ? `${h.cores} núcleos · ${h.threads} hilos` : "");
  if (!S.scanning) (h.gpus || []).forEach(g => cards += hwCard("Gráficos", g.name, g.vramMB ? Math.round(g.vramMB / 1024) + " GB de VRAM" : ""));
  if (!S.scanning) cards += hwCard("Memoria", (h.ramGB || "?") + " GB", [h.ramMTs ? h.ramMTs + " MT/s" : "", h.ramSticks ? h.ramSticks + " módulos" : ""].filter(Boolean).join(" · "));
  if (!S.scanning) (h.disks || []).forEach(d => cards += hwCard(d.system ? "Disco del sistema" : "Disco", d.model || d.type, `${d.type} · ${d.sizeGB} GB`));
  if (!S.scanning) cards += hwCard("Sistema", h.os || "Windows", h.build ? "compilación " + h.build : "");
  m.innerHTML = `
  <div class="top"><div><h2>Inicio</h2><p>Escaneamos tu equipo y solo te ofrecemos lo que le sirve. Todo se puede revertir.</p></div></div>
  ${banners()}${S.scanning ? '<div class="banner">Escaneando tu equipo… ya podés ir mirando las secciones; los perfiles se habilitan cuando termina.</div>' : ""}
  <section class="hero">
    <div class="ring" role="img" aria-label="${pct}% de las optimizaciones recomendadas aplicadas">
      <div class="num"><b>${pct}<small>%</small></b><span>${on} de ${total} optimizaciones recomendadas aplicadas</span></div>
      <div class="scale" aria-hidden="true">${Array.from({ length: 50 }, (_, i) => `<i class="${i < Math.round(pct / 2) ? "f" : ""}" style="--i:${i}"></i>`).join("")}</div>
    </div>
    <div class="hero-body">
      <div>
        <h3>${pct >= 100 ? "Tu PC ya está optimizada" : "Optimiza tu PC en un clic"}</h3>
        <p>El perfil recomendado aplica solo ajustes seguros para tu hardware. El competitivo suma ajustes de latencia para shooters y esports.</p>
      </div>
      <div class="btns">
        <button class="btn primary" id="pRec" ${S.admin && !S.scanning ? "" : "disabled"}>Aplicar perfil recomendado</button>
        <button class="btn" id="pComp" ${S.admin && !S.scanning ? "" : "disabled"}>Perfil competitivo</button>
        <button class="btn" id="pRp" ${S.admin ? "" : "disabled"}>Crear punto de restauración</button>
        <button class="btn" id="pScan">Volver a escanear</button>
        <button class="btn danger" id="pUndo" ${S.admin ? "" : "disabled"}>Restaurar todo</button>
      </div>
    </div>
  </section>
  <div class="grid">${cards}</div>`;
  $("#pRec").onclick = () => runProfile("recommended", "Perfil recomendado");
  $("#pComp").onclick = () => runProfile("competitive", "Perfil competitivo");
  $("#pRp").onclick = async () => {
    modal.open("Punto de restauración"); modal.log("Creando punto de restauración… puede tardar un momento.", "note");
    try { await api("restorepoint", {}); modal.log("Punto de restauración creado.", "ok"); } catch (e) { modal.log(e.message, "bad"); }
    modal.progress(1); modal.done();
  };
  $("#pUndo").onclick = undoAll;
  $("#pScan").onclick = async () => { toast("Escaneando…"); S.hardware = await api("rescan", {}); render(); };
}

async function undoAll() {
    const n = S.tweaks.filter(t => t.applied).length;
    if (!n) return toast("No hay nada aplicado por OverdriveTuner.");
    if (!await ask(`Se van a revertir los ${n} ajustes aplicados por OverdriveTuner y tu sistema quedará como antes de usarlo.\n\n¿Continuar?`, { title: "Restaurar todo", ok: "Restaurar", danger: true })) return;
    modal.open("Restaurar todo"); modal.log("Revirtiendo…", "note");
    try {
      const rs = await api("revertall", {});
      rs.forEach(r => modal.log((r.ok ? "Revertido: " : "No se pudo: ") + r.id + (r.errors ? " — " + r.errors.join("; ") : ""), r.ok ? "ok" : "bad"));
      if (rs.some(r => r.reboot)) modal.log("Reiniciá para completar la restauración.", "note");
    } catch (e) { modal.log(e.message, "bad"); }
    modal.progress(1); modal.done(); await refreshTweaks();
  }

/* ---------- tweaks ---------- */
function renderCategory(m, cat) {
  const c = S.categories.find(x => x.id === cat);
  if (!c) { view = "home"; return render(); }
  const q = query.trim().toLowerCase();
  const list = tweaksOf(cat).filter(t => !q || (t.name + " " + t.desc).toLowerCase().includes(q));
  const [a, n] = activeCount(cat);
  m.innerHTML = `
  <div class="top"><div><h2>${esc(c.name)}</h2><p>${esc(c.desc)}</p></div>
    <div class="search"><input id="q" type="search" placeholder="Buscar en esta sección" aria-label="Buscar" value="${esc(query)}"></div></div>
  ${banners()}
  ${cat === "apps" || cat === "debloat" ? '<div class="banner">Todo lo de esta sección se puede deshacer: apagá el interruptor o usá «Revertir los de esta sección». Cada cambio queda anotado en <b>Historial</b>.</div>' : ""}
  ${(typeof CAT_INFO !== "undefined" && CAT_INFO[cat]) ? `<p class="note">${esc(CAT_INFO[cat])}</p>` : ""}
  <div class="bar">
    <button class="btn" id="allOn" ${S.admin ? "" : "disabled"}>Aplicar todos los compatibles</button>
    <button class="btn" id="allOff" ${S.admin ? "" : "disabled"}>Revertir los de esta sección</button>
    <span class="sp">${a} de ${n} activos</span>
  </div>
  <div class="list" id="list">${list.map(rowHtml).join("") || '<p class="note">No hay resultados para esa búsqueda.</p>'}</div>`;
  const q$ = $("#q");
  q$.oninput = () => { query = q$.value; const pos = q$.selectionStart; renderCategory(m, cat); const n2 = $("#q"); n2.focus(); n2.setSelectionRange(pos, pos); };
  $("#list").querySelectorAll(".sw").forEach(b => b.onclick = () => toggle(b.dataset.id, b));
  $("#allOn").onclick = () => runIds(`Aplicar: ${c.name}`, tweaksOf(cat).filter(t => t.suitable && !t.active && t.risk < 2).map(t => t.id), "apply");
  $("#allOff").onclick = () => runIds(`Revertir: ${c.name}`, tweaksOf(cat).filter(t => t.active).map(t => t.id), "revert");
}

function rowHtml(t) {
  const riskTxt = ["Seguro", "Riesgo medio", "Avanzado"][t.risk];
  const meter = '<i></i><i></i><i></i>';
  const tags = (t.reboot ? '<span class="tag">Requiere reiniciar</span>' : "") +
    (t.active && !t.applied ? '<span class="tag">Ya activo en tu sistema</span>' : "") +
    (t.drift ? '<span class="tag bad">Windows lo deshizo: desactivalo y volvé a activarlo</span>' : "") +
    (!t.suitable ? `<span class="tag">No disponible: ${esc(t.whyNot)}</span>` : "");
  return `<div class="row ${t.active ? "on" : ""} ${t.suitable ? "" : "na"}">
    <div><h4>${esc(t.name)} <span class="risk r${t.risk}">${meter}${riskTxt}</span> ${tags}</h4>
      <p>${esc(t.desc)}</p>${t.warn ? `<div class="warn"><b>Ojo:</b>${esc(t.warn)}</div>` : ""}</div>
    <button class="sw" role="switch" aria-checked="${t.active}" aria-label="${esc(t.name)}" data-id="${esc(t.id)}" ${t.suitable && S.admin ? "" : "disabled"}></button>
  </div>`;
}

async function toggle(id, btn) {
  if (busyOp) return;
  const t = S.tweaks.find(x => x.id === id);
  const turningOn = !t.active;
  if (turningOn && t.risk >= 2 && !t.ac && !await ask(`${t.warn || "Este ajuste es avanzado."}\n\n¿Quieres aplicarlo de todos modos?`, { title: t.name, ok: "Aplicar de todos modos" })) return;
  let confirmN = 0;
  if (turningOn && t.ac) {
    if (!await ask(`«${t.name}» desactiva: ${t.ac}.\nLos anti-cheat de kernel lo exigen: si lo desactivás probablemente no puedas jugar partidas competitivas hasta volver a activarlo.\n\nPrimera confirmación: ¿querés continuar?`, { title: "Anti-cheat", ok: "Continuar", danger: true })) return;
    const w = await ask(`Para desactivar «${t.ac}» escribí DESACTIVAR (en inglés: DISABLE):`, { title: "Segunda confirmación", ok: "Desactivar", danger: true, word: ["DESACTIVAR", "DISABLE"] });
    if (!w) { toast("Cancelado: no se cambió nada."); return; }
    confirmN = 2;
  }
  busyOp = true; btn.classList.add("busy"); btn.disabled = true;
  try {
    if (turningOn && t.risk >= 2) { try { await api("restorepoint", {}); toast("Punto de restauración creado."); } catch (e) { /* Windows limita uno por día */ } }
    const r = await api(turningOn ? "apply" : "revert", { id, confirm: confirmN });
    if (!r.ok) throw new Error((r.errors || ["Error desconocido"]).join("\n"));
    toast((turningOn ? "Aplicado: " : "Revertido: ") + t.name + (r.reboot ? " (reinicia para completar)" : ""));
  } catch (e) { toast("No se pudo: " + e.message, 6000); }
  busyOp = false;
  await refreshTweaks();
}

async function refreshTweaks() { const st = await api("state"); S = st; render(); }

/* ---------- ejecución por lotes ---------- */
async function runIds(title, ids, mode, withRestore) {
  if (busyOp) return;
  if (!ids.length) { toast("No hay nada para " + (mode === "apply" ? "aplicar" : "revertir") + " aquí."); return; }
  busyOp = true; modal.open(title);
  let ok = 0, bad = 0, reboot = false;
  try {
    if (withRestore) {
      modal.log("Creando punto de restauración…", "note");
      try { await api("restorepoint", {}); modal.log("Punto de restauración creado.", "ok"); }
      catch (e) { modal.log("No se pudo crear el punto de restauración: " + e.message, "bad"); modal.log("Continuamos igual: cada cambio guarda su propia copia para revertirlo.", "note"); }
    }
    for (let i = 0; i < ids.length; i++) {
      const t = S.tweaks.find(x => x.id === ids[i]);
      modal.log((mode === "apply" ? "Aplicando " : "Revirtiendo ") + t.name + "…", "note");
      try {
        const r = await api(mode, { id: t.id });
        if (r.ok) { ok++; reboot = reboot || r.reboot; modal.log("  listo", "ok"); (r.notes || []).forEach(x => modal.log("  " + x, "note")); }
        else { bad++; (r.errors || []).forEach(x => modal.log("  " + x, "bad")); }
      } catch (e) { bad++; modal.log("  " + e.message, "bad"); }
      modal.progress((i + 1) / ids.length);
    }
  } finally {
    busyOp = false;
    modal.log(`Terminado: ${ok} correctos, ${bad} con error.`, bad ? "bad" : "ok");
    if (reboot) modal.log("Reinicia el equipo para completar algunos cambios.", "note");
    modal.done();
    try { S = await api("state"); } catch (e) { /* se reintentará al cerrar */ }
  }
}

async function runProfile(name, title) {
  const r = await api("profile", { name });
  if (!r.ids.length) { toast("Ese perfil ya está aplicado en tu equipo."); return; }
  const names = r.ids.map(id => S.tweaks.find(t => t.id === id).name);
  if (!await ask(`${title}: se aplicarán ${r.ids.length} ajustes compatibles con tu equipo.\nSe creará antes un punto de restauración y podrás revertir cada ajuste.\n\n¿Continuar?`, { title, ok: "Aplicar" })) return;
  runIds(title, r.ids, "apply", true);
  void names;
}

/* ---------- limpieza ---------- */
function fmtBytes(b) {
  if (b < 1024) return b + " B";
  const u = ["KB", "MB", "GB", "TB"]; let i = -1;
  do { b /= 1024; i++; } while (b >= 1024 && i < u.length - 1);
  return b.toFixed(b >= 100 ? 0 : 1) + " " + u[i];
}

function cleanRow(x) {
  return `<tr>
    <td style="width:34px"><input type="checkbox" class="ck" data-id="${esc(x.id)}" aria-label="${esc(x.name)}" ${x.safe ? "checked" : ""}></td>
    <td><b>${esc(x.name)}</b><div class="cmd">${esc(x.desc)}</div></td>
    <td class="num">${x.files} archivos</td><td class="num"><b>${fmtBytes(x.bytes)}</b></td></tr>`;
}

async function renderClean(m) {
  m.innerHTML = `<div class="top"><div><h2>Limpieza</h2><p>Libera memoria RAM y espacio en disco: borra archivos temporales y cachés que Windows y los juegos vuelven a crear solos.</p></div></div><p class="note">Calculando…</p>`;
  let info;
  try { info = await api("clean/scan"); } catch (e) { m.innerHTML += `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "clean") return;
  const total = info.reduce((a, x) => a + x.bytes, 0);
  m.innerHTML = `
  <div class="top"><div><h2>Limpieza</h2><p>Libera memoria RAM y espacio en disco: borra archivos temporales y cachés que Windows y los juegos vuelven a crear solos.</p></div></div>
  ${banners()}
  <h3 style="margin-top:0">Memoria RAM</h3>
  <div class="list"><div class="row"><div><h4>Liberar memoria RAM</h4><p>Como «Acelerar» de Microsoft PC Manager: le pide a Windows que devuelva la memoria que los programas abiertos no están usando. Es temporal y los programas pueden tardar un instante en volver a responder. Si hay un juego abierto, no se toca el juego.</p><div id="ramRes" class="note" aria-live="polite"></div></div><button class="btn primary" id="ramBtn" ${S.admin ? "" : "disabled"}>Liberar RAM</button></div></div>
  <h3>Archivos temporales</h3>
  <table class="table"><tbody>${info.filter(x => !x.deep).map(cleanRow).join("")}</tbody></table>
  ${info.some(x => x.deep) ? `<h3 style="margin-top:22px">Limpieza profunda</h3><p class="note">Cachés de navegadores y apps. No borra contraseñas, cookies ni historial; la primera carga después puede ser un poco más lenta. Vienen desmarcadas.</p><table class="table"><tbody>${info.filter(x => x.deep).map(cleanRow).join("")}</tbody></table>` : ""}
  <div class="bar" style="margin-top:16px"><button class="btn primary" id="doClean">Limpiar lo seleccionado</button><span class="sp">Se pueden liberar hasta ${fmtBytes(total)}</span></div>`;
  $("#ramBtn").onclick = async () => {
    const b = $("#ramBtn"); b.disabled = true; $("#ramRes").textContent = "Liberando…";
    try {
      const r = await api("memory/free", {}), gb = v => (v / 1024).toFixed(1) + " GB", gain = r.afterMB - r.beforeMB;
      const L = t => `<div>${t}</div>`;
      $("#ramRes").innerHTML = L(`Memoria libre antes: ${gb(r.beforeMB)}`) + L(`Memoria libre después: ${gb(r.afterMB)} (${gain >= 0 ? "+" : ""}${gain} MB)`) + L(`Se recortaron ${r.trimmed} programas.`) +
        (r.skippedGame ? L(`No se tocó ${esc(r.skippedGame)}.`) : "") + (r.standby ? L("Se vació la memoria en espera.") : "") + (r.note ? L(esc(r.note)) : "");
    } catch (e) { $("#ramRes").textContent = e.message; }
    b.disabled = !S.admin;
  };
  $("#doClean").onclick = async () => {
    const ids = [...m.querySelectorAll(".ck:checked")].map(c => c.dataset.id);
    if (!ids.length) return toast("Selecciona qué limpiar.");
    modal.open("Limpiando"); modal.log("Borrando archivos…", "note");
    try {
      const res = await api("clean/run", { ids });
      let freed = 0; res.forEach(r => { freed += r.freed; modal.log(`${info.find(i => i.id === r.id).name}: ${fmtBytes(r.freed)} liberados` + (r.skipped ? ` (${r.skipped} en uso, omitidos)` : ""), "ok"); });
      modal.log(`Total liberado: ${fmtBytes(freed)}`, "ok");
    } catch (e) { modal.log(e.message, "bad"); }
    modal.progress(1); modal.done();
  };
}

/* ---------- inicio de windows ---------- */
async function renderStartup(m) {
  m.innerHTML = `<div class="top"><div><h2>Inicio de Windows</h2><p>Programas que arrancan con tu sesión. Desactiva los que no necesites: no se borran, puedes volver a activarlos.</p></div></div><p class="note">Leyendo…</p>`;
  let items;
  try { items = await api("startup"); } catch (e) { m.innerHTML += `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "startup") return;
  m.innerHTML = `
  <div class="top"><div><h2>Inicio de Windows</h2><p>Programas que arrancan con tu sesión. Desactiva los que no necesites: no se borran, puedes volver a activarlos.</p></div></div>
  ${banners()}
  ${items.length ? `<table class="table"><tbody>${items.map((x, i) => `<tr>
    <td><b>${esc(x.name)}</b> <span class="tag">${esc(x.scope)}</span><div class="cmd">${esc(x.command)}</div></td>
    <td style="width:70px;text-align:right"><button class="sw" role="switch" aria-checked="${x.enabled}" aria-label="${esc(x.name)}" data-i="${i}"></button></td></tr>`).join("")}</tbody></table>`
    : '<p class="note">No se encontraron programas de inicio en el registro.</p>'}`;
  m.querySelectorAll(".sw").forEach(b => b.onclick = async () => {
    const x = items[+b.dataset.i], want = b.getAttribute("aria-checked") !== "true";
    b.disabled = true;
    try { await api("startup/set", { source: x.source, name: x.name, enabled: want }); x.enabled = want; b.setAttribute("aria-checked", String(want)); }
    catch (e) { toast("No se pudo cambiar: " + e.message, 6000); }
    b.disabled = false;
  });
}

/* ---------- fluidez ---------- */
function gameGuide() {
  const h = (S && S.hardware) || {}, amdGpu = (h.gpus || []).some(g => g.vendor === "amd");
  const mem = h.cpuVendor === "amd" ? "DOCP/EXPO" : h.cpuVendor === "intel" ? "XMP" : "XMP/DOCP";
  const g = [
  ["CS2 (dentro del juego)", "Multicore rendering: activado · V-Sync: desactivado · Sombras, efectos y sombreado: bajos · Oclusión ambiental: desactivada · FSR: desactivado (al estar limitado por CPU no suma). Es un juego que depende sobre todo del procesador y la memoria, no de la gráfica."],
  ["Límite de FPS (probar con el benchmark)", "Poner fps_max apenas por debajo de 240 (por ejemplo 237) y, si el monitor tiene FreeSync, activarlo, puede evitar cortes. Es una idea para probar, no una garantía: medilo con el benchmark antes y después."],
  ["Memoria", `Si el diagnóstico dice que la RAM no corre a su velocidad nominal, activar ${mem} en la BIOS es de lo que más ayuda.`],
  ["Segundo plano", "Cerrá navegadores, launchers y overlays antes de jugar. Abajo ves qué proceso usa más CPU."],
  ["Anti-cheat de kernel", "OverdriveTuner no toca Secure Boot, TPM, IOMMU, integridad de memoria ni las actualizaciones de seguridad. Lo único que las afecta está en «Avanzado» y pide doble confirmación."],
  ];
  if (amdGpu) g.splice(2, 0, ["Driver de AMD", "NO actives Anti-Lag+ en CS2: puede ser detectado como modificación y causar baneos VAC. Si querés probar, usá solo la opción Anti-Lag común o dejalo apagado."]);
  return g;
}
function biosGuide() {
  const h = (S && S.hardware) || {}, amd = h.cpuVendor === "amd", intel = h.cpuVendor === "intel";
  const dGpu = (h.gpus || []).some(g => !/graphics|uhd|iris|vega|hd/i.test(g.name || "") || /rx |rtx|gtx|arc a/i.test(g.name || ""));
  const g = [];
  g.push([amd ? "DOCP / EXPO (memoria)" : intel ? "XMP (memoria)" : "XMP / DOCP (memoria)", `Activá el perfil de memoria en la BIOS (${amd ? "en placas AMD se llama DOCP o EXPO" : intel ? "en placas Intel se llama XMP" : "se llama XMP en Intel y DOCP o EXPO en AMD"}). Sin esto la RAM corre más lenta de lo que compraste.`]);
  if (dGpu) g.push(["Resizable BAR", `Con una placa de video dedicada suele dar una mejora chica pero gratis. En la BIOS: Above 4G Decoding en Enabled y Re-Size BAR Support${amd ? " (en AMD también se llama Smart Access Memory)" : ""} en Enabled. Puede requerir actualizar la BIOS; hacelo solo si hace falta y siguiendo el manual.`]);
  g.push([amd ? "IOMMU y SVM" : intel ? "VT-d y VT-x" : "Virtualización / IOMMU", amd ? "Los anti-cheat de kernel piden IOMMU: en la BIOS aparece como «IOMMU» o «SVM Mode». Dejalo activado." : intel ? "Los anti-cheat de kernel piden IOMMU: en la BIOS de Intel se llama «VT-d» (y «VT-x» para virtualización). Dejalos activados." : "Los anti-cheat de kernel piden IOMMU: en la BIOS se llama «IOMMU» o «SVM» (AMD) o «VT-d» (Intel). Dejalo activado."]);
  g.push(["Secure Boot y TPM", `Los anti-cheat de kernel los exigen. El TPM en la BIOS se llama ${amd ? "«fTPM» o «AMD CPU fTPM»" : intel ? "«PTT» (Intel Platform Trust Technology)" : "«fTPM» (AMD) o «PTT» (Intel)"}: dejalo activado.`]);
  g.push(["Antes de tocar la BIOS", "Anotá cada cambio y hacelo de a uno. Si el PC no arranca, la opción «Load Optimized Defaults» lo deja como venía. Los nombres exactos cambian según el fabricante de la placa."]);
  return g;
}

const stCls = { ok: "ok", warn: "warn", bad: "bad", info: "note" };
const stIc = s => `<span class="st ${s}" aria-hidden="true"></span>`;

async function renderFluidez(m) {
  const head = `<div class="top"><div><h2>Diagnóstico</h2><p>Revisa lo que más causa microcortes y falta de fluidez, sobre todo en monitores de alta frecuencia. Nada se cambia solo.</p></div></div>`;
  m.innerHTML = head + '<p class="note">Revisando…</p>';
  let d;
  try { d = await api("diag"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "fluidez") return;
  m.innerHTML = head + banners() +
    `<div class="banner ${d.bad ? "bad" : ""}">${d.bad ? d.bad + " problema(s) importante(s)" : "Sin problemas importantes"}${d.warn ? " · " + d.warn + " aviso(s)" : ""} · monitor objetivo: ${d.targetHz ? d.targetHz + " Hz" + (d.manual ? "" : " (detectado)") : "sin detectar"}</div>` +
    `<div class="list">${d.checks.map(c => `<div class="row ${c.status === "ok" ? "on" : ""}"><div><h4>${stIc(c.status)}${esc(c.title)}</h4><p>${esc(c.detail)}</p></div>${c.action ? `<button class="btn" data-a="${esc(c.action)}">${esc(c.actionLabel)}</button>` : ""}</div>`).join("")}</div>
    <div class="bar" style="margin:16px 0"><label>Frecuencia de tu monitor (Hz): <input id="hz" type="number" min="30" max="1000" value="${d.targetHz || ""}" style="width:80px"></label><button class="btn" id="hzSave">Guardar</button>${d.manual && d.detectedHz ? `<button class="btn" id="hzAuto">Usar el detectado (${d.detectedHz} Hz)</button>` : ""}<button class="btn" id="dgAgain">Volver a revisar</button></div>
    <h3>Qué está usando tu procesador</h3><div id="procs"><button class="btn" id="procGo">Medir ahora (2 segundos)</button></div>
    <h3 style="margin-top:22px">Herramientas</h3>
    <div class="btns"><button class="btn" id="tNet">Probar red y ping (15 s)</button><button class="btn" id="tDns">Probar DNS</button><button class="btn" id="tNetMon">Quién usa la red</button><button class="btn" id="tDisk">Estado de los discos</button><button class="btn" id="tVid">Revisar video de CS2</button><button class="btn" id="tRep">Descargar informe</button></div>
    <div id="tOut" style="margin-top:12px"></div>
    <h3 style="margin-top:22px">Guía de BIOS${(S.hardware && S.hardware.board) ? " · " + esc(S.hardware.board) : ""}</h3><p class="note">Armada según tu procesador y tu placa de video detectados; los nombres exactos de cada opción dependen del fabricante de tu placa madre.</p><div class="list">${biosGuide().map(g => `<div class="row"><div><h4>${esc(g[0])}</h4><p>${esc(g[1])}</p></div></div>`).join("")}</div>
    <h3 style="margin-top:22px">Guía para jugar fluido</h3><div class="list">${gameGuide().map(g => `<div class="row"><div><h4>${esc(g[0])}</h4><p>${esc(g[1])}</p></div></div>`).join("")}</div>`;
  m.querySelectorAll("[data-a]").forEach(b => b.onclick = () => api("diag/action", { action: b.dataset.a }).catch(e => toast(e.message, 5000)));
  $("#hzSave").onclick = async () => { try { await api("settings", { targetHz: +$("#hz").value }); renderFluidez(m); } catch (e) { toast(e.message, 5000); } };
  if ($("#hzAuto")) $("#hzAuto").onclick = async () => { try { await api("settings", { hzAuto: true }); renderFluidez(m); } catch (e) { toast(e.message, 5000); } };
  $("#dgAgain").onclick = async () => { await api("diag?fresh=1"); renderFluidez(m); };
  const tout = $("#tOut");
  const rowsOf = (items, f) => `<div class="list">${items.map(f).join("")}</div>`;
  const st = x => `<span class="tag">${esc(({ ok: "bien", warn: "aviso", bad: "problema", info: "info", error: "sin datos" })[x] || x)}</span>`;
  const runTool = async (btn, label, fn) => { btn.disabled = true; tout.innerHTML = `<p class="note">${label}</p>`; try { tout.innerHTML = await fn(); } catch (e) { tout.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; } btn.disabled = false; };
  $("#tNet").onclick = e => runTool(e.target, "Midiendo (unos 15 segundos)…", async () => rowsOf(await api("net"), r => `<div class="row"><div><h4>${esc(r.name)} (${esc(r.host)}) ${st(r.status)}</h4><p>${r.sent ? `Promedio ${r.avg} ms · mín ${r.min} · máx ${r.max} · variación ${r.jitter} ms · pérdida ${r.lossPct}%. ` : ""}${esc(r.detail)}</p></div></div>`));
  $("#tDisk").onclick = e => runTool(e.target, "Leyendo discos…", async () => { const ds = await api("disks"); return ds.length ? rowsOf(ds, d => diskRow(d, st)) : '<p class="note">Windows no devolvió información de los discos.</p>'; });
  $("#tDns").onclick = e => runTool(e.target, "Midiendo servidores DNS (unos 10 segundos)…", async () => dnsHtml(await api("dnsbench")));
  $("#tNetMon").onclick = e => runTool(e.target, "Leyendo conexiones…", async () => netMonHtml(await api("netmon")));
  tout.addEventListener("click", dnsClick);
  $("#tVid").onclick = e => runTool(e.target, "Buscando cs2_video.txt…", async () => { const v = await api("cs2video"); return `<p class="note">${esc(v.message)}</p>` + rowsOf(v.findings, f => `<div class="row"><div><h4>${esc(f.key)}: ${esc(f.value)} ${st(f.status)}</h4><p>${esc(f.advice)}</p></div></div>`); });
  $("#tRep").onclick = async () => { try { const r = await fetch("/api/report", { headers: { "X-Token": window.TOKEN } }); if (!r.ok) throw new Error("Error " + r.status); const a = document.createElement("a"); a.href = URL.createObjectURL(await r.blob()); a.download = "informe-overdrivetuner.html"; a.click(); } catch (err) { toast(err.message, 5000); } };
  $("#procGo").onclick = async () => {
    const box = $("#procs"); box.innerHTML = '<p class="note">Midiendo…</p>';
    try {
      const ps = await api("procs");
      box.innerHTML = ps.length ? `<table class="table"><tbody>${ps.map((p, i) => `<tr><td><b>${esc(p.name)}</b></td><td class="num">${p.cpu}% de un núcleo</td><td class="num">${p.mb} MB</td><td style="text-align:right">${p.safe ? `<button class="btn danger" data-i="${i}">Cerrar</button>` : '<span class="tag">protegido</span>'}</td></tr>`).join("")}</tbody></table>` : '<p class="note">Nada usa CPU de forma notable.</p>';
      box.querySelectorAll("[data-i]").forEach(b => b.onclick = async () => {
        const p = ps[+b.dataset.i];
        if (!await ask(`Perdés lo que no hayas guardado en ese programa.`, { title: `Cerrar «${p.name}»`, ok: "Cerrar programa", danger: true })) return;
        try { await api("procs/kill", { pid: p.pid, name: p.name }); toast("Cerrado: " + p.name); b.closest("tr").remove(); } catch (e) { toast(e.message, 6000); }
      });
    } catch (e) { box.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; }
  };
}

/* ---------- benchmark ---------- */
async function renderBench(m) {
  const head = `<div class="top"><div><h2>Benchmark</h2><p>Pegá el resultado del benchmark de CS2 (Afterburner/FPS Benchmark). Marcá en qué estado estaba el PC y compará con varias corridas: una sola no alcanza, el ruido entre corridas es de ±20 FPS.</p></div></div>`;
  let d;
  try { d = await api("bench"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "bench") return;
  const f = x => (+x).toFixed(1);
  m.innerHTML = head + banners() + `
  <div class="row"><div style="width:100%">
    <textarea id="bTxt" rows="6" style="width:100%" placeholder="Pegá acá el texto del benchmark…"></textarea>
    <div class="bar" style="margin-top:8px"><label>Estado: <input id="bLbl" maxlength="60" placeholder="ej. antes / optimizado / sin tweaks de red"></label>
      <label><input type="checkbox" id="bBase"> es el «antes de optimizar»</label><button class="btn primary" id="bAdd">Guardar corridas</button></div></div></div>
  ${d.groups.length ? `<h3>Resumen por estado</h3><table class="table"><thead><tr><th>Estado</th><th>Corridas</th><th>FPS prom.</th><th>Rango</th><th>Desvío</th><th>1% low</th><th>0.1% low</th></tr></thead><tbody>${d.groups.map(g => `<tr><td><b>${esc(g.label)}</b></td><td class="num">${g.n}</td><td class="num"><b>${f(g.mean)}</b></td><td class="num">${f(g.min)}–${f(g.max)}</td><td class="num">±${f(g.sd)}</td><td class="num">${f(g.low1)}</td><td class="num">${f(g.low01)}</td></tr>`).join("")}</tbody></table>
  ${d.verdicts.map(v => `<div class="banner ${v.real && v.diffPct < 0 ? "bad" : ""}"><b>${esc(v.a)} → ${esc(v.b)}: ${v.diffPct > 0 ? "+" : ""}${f(v.diffPct)}%</b><br>${esc(v.text)}</div>`).join("")}` : '<p class="note">Todavía no cargaste corridas.</p>'}
  ${d.entries.length ? `<h3>Corridas</h3><table class="table"><tbody>${d.entries.map(e => `<tr><td>${esc(e.run.label || "sin etiqueta")}${e.baseline ? ' <span class="tag">base</span>' : ""}</td><td class="num">${esc(e.run.date)}</td><td class="num">${f(e.run.avg)} FPS</td><td class="num">1%: ${f(e.run.low1)}</td><td style="text-align:right"><button class="btn" data-id="${e.id}">Borrar</button></td></tr>`).join("")}</tbody></table><button class="btn danger" id="bAll" style="margin-top:10px">Borrar todo</button>` : ""}`;
  $("#bAdd").onclick = async () => {
    try { await api("bench/add", { text: $("#bTxt").value, label: $("#bLbl").value.trim(), baseline: $("#bBase").checked }); renderBench(m); }
    catch (e) { toast(e.message, 7000); }
  };
  m.querySelectorAll("[data-id]").forEach(b => b.onclick = async () => { await api("bench/delete", { id: +b.dataset.id }); renderBench(m); });
  renderSessions(m);
  renderBisect(m);
  const all = $("#bAll"); if (all) all.onclick = async () => { if (await ask("¿Borrar todas las corridas guardadas?", { title: "Borrar corridas", ok: "Borrar todo", danger: true })) { await api("bench/delete", { all: true }); renderBench(m); } };
}

/* ---------- sesiones medidas por el overlay + búsqueda del ajuste culpable ---------- */
function drawFrames(cv, frames) {
  const c = cv.getContext("2d"), W = cv.width, H = cv.height;
  c.clearRect(0, 0, W, H);
  if (!frames || frames.length < 2) return;
  const sorted = [...frames].sort((a, b) => a - b), med = sorted[sorted.length >> 1], top = Math.max(med * 4, 14);
  c.fillStyle = "#0a0a0a"; c.fillRect(0, 0, W, H);
  const bw = Math.max(1, W / frames.length);
  frames.forEach((f, i) => {
    const h = Math.min(f, top) / top * (H - 6);
    c.fillStyle = f > 2.5 * med && f - med > 4 ? "#ff5f78" : f > 1.6 * med && f - med > 2 ? "#ffc857" : "#c4c4c4";
    c.fillRect(i * W / frames.length, H - h, bw, h);
  });
}

async function renderSessions(m) {
  const box = document.createElement("div"); box.style.marginTop = "26px"; m.appendChild(box);
  let list;
  try { list = await api("sessions"); } catch (e) { return; }
  box.innerHTML = `<h3>Corridas medidas con el overlay</h3><p class="note">Grabalas desde «Overlay en juego». Se miden cuadro por cuadro, así que se ven los microcortes. Importá una al benchmark para compararla.</p>` +
    (list.length ? `<table class="table"><tbody>${list.map(x => `<tr><td><b>${esc(x.label || "sin etiqueta")}</b> <span class="tag">${esc(x.at.slice(0, 16).replace("T", " "))}</span></td><td class="num">${(+x.run.avg).toFixed(1)} FPS</td><td class="num">1%: ${(+x.run.low1).toFixed(1)}</td><td class="num">${x.spikes} picos</td><td style="text-align:right"><button class="btn" data-v="${x.id}">Ver</button> <button class="btn" data-i="${x.id}" ${x.imported ? "disabled" : ""}>${x.imported ? "Importada" : "Importar"}</button> <button class="btn danger" data-d="${x.id}">Borrar</button></td></tr>`).join("")}</tbody></table><div id="sesView"></div>` : '<p class="note">Todavía no grabaste ninguna.</p>');
  box.querySelectorAll("[data-v]").forEach(b => b.onclick = async () => {
    const x = await api("sessions/get?id=" + b.dataset.v);
    $("#sesView").innerHTML = `<p class="note">Cada barra es un tramo de cuadros; rojo = pico (microcorte), amarillo = tirón leve. ${x.spikes} picos en ${(+x.run.seconds).toFixed(0)} s.</p><canvas id="cv" width="1000" height="150" style="width:100%;border-radius:10px"></canvas>`;
    drawFrames($("#cv"), x.frames);
    showMicrocuts(x.id);
  });
  box.querySelectorAll("[data-i]").forEach(b => b.onclick = async () => { try { await api("sessions/import", { id: b.dataset.i }); renderBench(m); } catch (e) { toast(e.message, 5000); } });
  box.querySelectorAll("[data-d]").forEach(b => b.onclick = async () => { await api("sessions/delete", { id: b.dataset.d }); renderBench(m); });
}

async function renderBisect(m) {
  const box = document.createElement("div"); box.style.marginTop = "26px"; m.appendChild(box);
  let d;
  try { d = await api("bisect"); } catch (e) { return; }
  const st = d.state, nm = id => d.names[id] || id;
  let body = "";
  const active = st.active, done = st.done;
  if (active) {
    body = `<p>Ronda ${st.round}: <b>se revirtieron ${st.tested.length} ajustes</b> de ${st.suspects.length} sospechosos. Medí con las mismas apps abiertas (podés grabar con el overlay) y contestá:</p>
      <ul>${st.tested.map(id => `<li>${esc(nm(id))}</li>`).join("")}</ul>
      <div class="btns"><button class="btn primary" id="bGood">El problema desapareció</button><button class="btn" id="bBad">Sigue igual</button><button class="btn danger" id="bCancel">Cancelar y volver a aplicar todo</button></div>`;
  } else if (done) {
    body = `<div class="banner ${st.culprit ? "" : "warn"}">${esc(st.message)}${st.culprit ? `<br><b>${esc(nm(st.culprit))}</b>` : ""}</div><button class="btn" id="bNew">Empezar otra búsqueda</button>`;
  } else {
    body = `<p class="note">Si después de optimizar rindió peor de verdad (el benchmark dice «empeora de verdad»), esta herramienta revierte la mitad de tus ajustes, vos medís, y se repite hasta encontrar cuál es. Tenés ${d.available} ajustes aplicados que se pueden probar. Si el cambio era ruido, lo va a decir.</p><button class="btn primary" id="bStart" ${d.available < 2 || !S.admin ? "disabled" : ""}>Buscar el ajuste que empeora el rendimiento</button>`;
  }
  box.innerHTML = `<h3>Buscar el ajuste culpable</h3>${body}${d.errors && d.errors.length ? `<div class="banner bad">${d.errors.map(esc).join("<br>")}</div>` : ""}`;
  const go = (path, b) => async () => { try { await api(path, b); await refreshTweaks(); } catch (e) { toast(e.message, 6000); } };
  const bind = (id, path, b) => { const el = $(id); if (el) el.onclick = go(path, b); };
  bind("#bStart", "bisect/start", {}); bind("#bNew", "bisect/start", {});
  bind("#bGood", "bisect/answer", { good: true }); bind("#bBad", "bisect/answer", { good: false });
  bind("#bCancel", "bisect/answer", { cancel: true });
}

/* ---------- overlay ---------- */
let ovTimer = null;
const OVL_ITEMS = [["showFps", "FPS"], ["showLow", "1% low"], ["showGraph", "Gráfico de cuadros"], ["showCpu", "CPU"], ["showGpu", "GPU"], ["showRam", "RAM"], ["showVram", "VRAM"], ["showTemps", "Temperaturas"]];

function ovPreview(cfg, st) {
  const sm = (st && st.sample) || { cpu: 41, gpu: 70, gpuTemp: 54, cpuTemp: -1, ramUsedMb: 11800 };
  const fps = st && st.fps ? Math.round(st.fps) : 238, low = st && st.low1 ? Math.round(st.low1) : 126;
  const t = v => v > 0 ? `<i class="${v >= 85 ? "bd" : v >= 70 ? "wn" : "gd"}">${Math.round(v)}°</i>` : `<i class="dm">—</i>`;
  const bars = Array.from({ length: 40 }, (_, i) => { const sp = i === 27 ? 26 : i === 11 ? 14 : 5 + (i * 7 % 3); return `<b class="${sp > 20 ? "bd" : sp > 10 ? "wn" : "gd"}" style="height:${sp / 26 * 100}%"></b>`; }).join("");
  const rows = [];
  if (cfg.showCpu) rows.push(`<div><span>CPU</span><em>${Math.round(sm.cpu)}%</em>${cfg.showTemps ? t(sm.cpuTemp) : ""}</div>`);
  if (cfg.showGpu) rows.push(`<div><span>GPU</span><em>${Math.round(sm.gpu)}%</em>${cfg.showTemps ? t(sm.gpuTemp) : ""}</div>`);
  if (cfg.showVram) rows.push(`<div><span>VRAM</span><em>${((sm.gpuVramMb || 0) / 1024).toFixed(1)} GB</em></div>`);
  if (cfg.showRam) rows.push(`<div><span>RAM</span><em>${(sm.ramUsedMb / 1024).toFixed(1)} GB</em></div>`);
  if (cfg.layout === "line") {
    const parts = []; if (cfg.showFps) parts.push(`<em>${fps} FPS</em>`);
    rows.forEach(r => parts.push(r.replace(/<\/?div>/g, "")));
    return `<div class="ovp line" style="opacity:${cfg.opacity};font-size:${13 * cfg.scale}px">${parts.join(" ")}</div>`;
  }
  return `<div class="ovp card" style="opacity:${cfg.opacity};font-size:${13 * cfg.scale}px">
    ${cfg.showFps ? `<div class="top2"><strong>${fps}</strong><small>FPS</small>${cfg.showLow ? `<small class="r">1% ${low}</small>` : ""}</div>` : ""}
    ${cfg.showGraph ? `<div class="spk">${bars}</div>` : ""}${rows.join("")}</div>`;
}

function ovStatus(d) {
  const st = d.state, a = d.running;
  const sm = st.sample || {};
  const tmp = v => v > 0 ? Math.round(v) + " °C" : "no disponible";
  return `<div class="row ${a ? "on" : ""}"><div><h4>${a ? "Overlay abierto" : "Overlay cerrado"} ${a && st.gameRunning ? '<span class="tag">' + esc(st.game) + " detectado</span>" : a ? '<span class="tag">esperando ' + (st.game ? esc(st.game) : "un juego") + "</span>" : ""}</h4>
    <p>${a ? `FPS ${Math.round(st.fps || 0)} · 1% low ${Math.round(st.low1 || 0)} · CPU ${Math.round(sm.cpu || 0)}% (hilo más cargado ${Math.round(sm.cpuThread || 0)}%) · GPU ${Math.round(sm.gpu || 0)}% ${tmp(sm.gpuTemp)} · RAM ${((sm.ramUsedMb || 0) / 1024).toFixed(1)} GB · temp. CPU ${tmp(sm.cpuTemp)}` : "Abrilo para ver FPS y uso mientras jugás."}</p>
    ${a && st.verdict && st.verdict.kind !== "none" ? `<p><b>Qué limita tus FPS:</b> ${esc(st.verdict.text)}</p>` : ""}
    ${a && st.message ? `<p class="note">${esc(st.message)}</p>` : ""}</div>
    <button class="btn ${a ? "" : "primary"}" id="ovToggle" ${S.admin ? "" : "disabled"}>${a ? "Cerrar overlay" : "Abrir overlay"}</button></div>`;
}

async function renderOverlay(m) {
  clearTimeout(ovTimer);
  const head = `<div class="top"><div><h2>Overlay en juego</h2><p>Un panel mínimo con FPS, CPU, GPU y RAM encima del juego. No se inyecta en el juego (es una ventana aparte que deja pasar los clics), por eso no molesta a los anti-cheat de kernel ni a VAC. Los atajos de teclado se cambian en Configuración.</p></div><div class="btns"><button class="btn" id="fhBtn">Cómo se miden los FPS</button><button class="btn" id="ovGuideBtn">Entender los números</button></div></div>`;
  let d;
  try { d = await api("overlay"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  const cfg = d.config;
  const save = async () => { try { Object.assign(cfg, await api("overlay/config", cfg)); } catch (e) { toast(e.message, 5000); } $("#ovPrev").innerHTML = ovPreview(cfg, d.state); };
  m.innerHTML = head + banners() + `<div id="ovStatus">${ovStatus(d)}</div>
  <div class="ovgrid"><div><h3>Vista previa</h3><div class="ovstage" id="ovPrev">${ovPreview(cfg, d.running ? d.state : null)}</div><p class="note">Es una aproximación: el diseño real lo dibuja Windows y puede verse un poco distinto.</p></div>
  <div><h3>Apariencia</h3><div class="list">
   <div class="row"><div><h4>Esquina</h4></div><select id="cCorner">${[["tl", "Arriba izquierda"], ["tr", "Arriba derecha"], ["bl", "Abajo izquierda"], ["br", "Abajo derecha"]].map(([v, n]) => `<option value="${v}" ${cfg.corner === v ? "selected" : ""}>${n}</option>`).join("")}</select></div>
   <div class="row"><div><h4>Diseño</h4></div><select id="cLayout"><option value="card" ${cfg.layout === "card" ? "selected" : ""}>Tarjeta</option><option value="line" ${cfg.layout === "line" ? "selected" : ""}>Una línea</option></select></div>
   <div class="row"><div><h4>Frecuencia de actualización</h4><p>Cuántas veces por segundo se actualiza el número de FPS. Más rápido sigue mejor los cambios.</p></div><select id="cUpd">${[[50, "20 por segundo"], [100, "10 por segundo"], [250, "4 por segundo"], [500, "2 por segundo"]].map(([v, n]) => `<option value="${v}" ${cfg.updateMs === v ? "selected" : ""}>${n}</option>`).join("")}</select></div>
   <div class="row"><div><h4>Tamaño</h4></div><input type="range" id="cScale" min="0.8" max="1.8" step="0.1" value="${cfg.scale}"></div>
   <div class="row"><div><h4>Transparencia</h4></div><input type="range" id="cOp" min="0.3" max="1" step="0.05" value="${cfg.opacity}"></div>
   <div class="row"><div><h4>Qué mostrar</h4><p style="display:flex;flex-wrap:wrap;gap:12px">${OVL_ITEMS.map(([k, n]) => `<label><input type="checkbox" data-k="${k}" ${cfg[k] ? "checked" : ""}> ${n}</label>`).join("")}</p></div></div>
   <div class="row"><div><h4>Avisar si la PC se calienta</h4><p>Cuando el procesador o la gráfica pasan el límite, el valor se pone en rojo y el overlay muestra un aviso de texto (sin sonido). La de la gráfica funciona siempre; la del procesador solo si Core Temp está instalado.</p></div><input type="checkbox" data-k="alertOn" aria-label="Avisar si la PC se calienta" ${cfg.alertOn ? "checked" : ""}></div>
   <div class="row"><div><h4>Temperatura del procesador</h4><p>Windows no la entrega en la mayoría de los procesadores. El overlay la lee de Core Temp, que se abre solo en segundo plano si lo tenés instalado. Solo lee: no se mete en el juego. Core Temp es gratis y está en «Instalar programas».</p></div></div>
   <div class="row"><div><h4>Límites de temperatura</h4><p>En grados Celsius, entre 60 y 110.</p></div><div class="bar"><label>CPU <input type="number" id="cAlC" min="60" max="110" value="${cfg.alertCpu}" style="width:70px"></label><label>GPU <input type="number" id="cAlG" min="60" max="110" value="${cfg.alertGpu}" style="width:70px"></label></div></div>
   <div class="row"><div><h4>Detectar el juego automáticamente</h4><p>Reconoce solo qué juego abriste (los conocidos, los que agregues abajo o cualquier programa a pantalla completa) y empieza a medir. No hace falta escribir nada.</p></div><input type="checkbox" data-k="autoGame" aria-label="Detectar el juego automáticamente" ${cfg.autoGame ? "checked" : ""}></div>
   <div class="row"><div><h4>Mostrar solo mientras juego</h4><p>El panel aparece al abrir un juego y se esconde cuando lo cerrás.</p></div><input type="checkbox" data-k="onlyInGame" aria-label="Mostrar solo mientras juego" ${cfg.onlyInGame ? "checked" : ""}></div>
   <div class="row"><div><h4>Abrir el overlay al iniciar OverdriveTuner</h4><p>Queda funcionando en segundo plano mientras el programa esté abierto (también en la bandeja).</p></div><input type="checkbox" data-k="autoOpen" aria-label="Abrir el overlay al iniciar OverdriveTuner" ${cfg.autoOpen ? "checked" : ""}></div>
   <div class="row"><div><h4>Carpetas de juegos adicionales</h4><p>Una por línea. Todo programa con ventana que esté dentro de estas carpetas cuenta como juego. Los de Steam y Epic se reconocen solos, en cualquier disco; esto es para juegos instalados en otro lado, por ejemplo D:\Juegos.</p><textarea id="cDirs" rows="3" style="width:100%" placeholder="D:\Juegos">${esc((cfg.gameDirs || []).join("\n"))}</textarea></div></div>
   <div class="row"><div><h4>Juego a medir</h4><p>Solo se usa si apagás la detección automática. Nombre del programa, por ejemplo cs2.exe</p></div><input type="text" id="cGame" value="${esc(cfg.gameExe)}" style="width:150px"></div>
  </div></div></div>
  <h3 style="margin-top:22px">Grabar una corrida</h3>
  <div class="row"><div><p>Poné un nombre que describa el estado del PC (por ejemplo «antes» o «optimizado»), tocá Grabar, jugá el mismo tramo o benchmark y tocá Detener. Hacé 3 corridas por estado. Después se importan en «Benchmark».</p>
  <div class="bar"><input type="text" id="recLbl" placeholder="antes / optimizado…" style="width:200px"><button class="btn primary" id="recStart" ${d.running ? "" : "disabled"}>Grabar</button><button class="btn" id="recStop" ${d.running ? "" : "disabled"}>Detener y guardar</button></div></div></div>
  <h3 style="margin-top:22px">Perfiles por juego</h3><div class="list" id="ovGames"></div>
  <div class="bar" style="margin-top:10px"><input type="text" id="newGame" placeholder="otro juego, ej. eurotrucks2.exe" style="width:240px"><button class="btn" id="addGame">Agregar juego</button></div>`;
  $("#ovGuideBtn").onclick = openOverlayGuide;
  $("#fhBtn").onclick = openFpsHelp;
  const toggle = $("#ovToggle");
  toggle.onclick = async () => { try { await api(d.running ? "overlay/stop" : "overlay/start", {}); toast(d.running ? "Cerrando overlay…" : "Abriendo overlay…"); setTimeout(() => renderOverlay(m), 1800); } catch (e) { toast(e.message, 6000); } };
  $("#cCorner").onchange = e => { cfg.corner = e.target.value; save(); };
  $("#cUpd").onchange = e => { cfg.updateMs = +e.target.value; save(); };
  $("#cLayout").onchange = e => { cfg.layout = e.target.value; save(); };
  $("#cScale").oninput = e => { cfg.scale = +e.target.value; $("#ovPrev").innerHTML = ovPreview(cfg, d.state); };
  $("#cScale").onchange = save;
  $("#cOp").oninput = e => { cfg.opacity = +e.target.value; $("#ovPrev").innerHTML = ovPreview(cfg, d.state); };
  $("#cOp").onchange = save;
  $("#cAlC").onchange = e => { cfg.alertCpu = +e.target.value; save(); };
  $("#cAlG").onchange = e => { cfg.alertGpu = +e.target.value; save(); };
  $("#cDirs").onchange = e => { cfg.gameDirs = e.target.value.split("\n").map(x => x.trim()).filter(Boolean); save(); };
  $("#cGame").onchange = e => { cfg.gameExe = e.target.value.trim(); save(); };
  m.querySelectorAll("[data-k]").forEach(c => c.onchange = () => { cfg[c.dataset.k] = c.checked; save(); });
  $("#recStart").onclick = async () => { try { await api("overlay/record", { action: "start", label: $("#recLbl").value }); toast("Grabando…"); } catch (e) { toast(e.message, 5000); } };
  $("#recStop").onclick = async () => { try { await api("overlay/record", { action: "stop" }); toast("Guardando sesión… mirala en Benchmark."); } catch (e) { toast(e.message, 5000); } };
  // perfiles por juego
  const renderGames = () => {
    $("#ovGames").innerHTML = cfg.games.map((g, i) => `<div class="row"><div><h4>${esc(g.exe)}</h4>
      <p style="display:flex;flex-wrap:wrap;gap:12px;align-items:center"><label><input type="checkbox" data-gp="${i}" ${g.priority ? "checked" : ""}> Prioridad alta al abrir</label>
      <label>Plan de energía: <select data-gpl="${i}"><option value="">no cambiar</option>${d.plans.map(p => `<option value="${esc(p.guid)}" ${g.powerPlan === p.guid ? "selected" : ""}>${esc(p.name)}</option>`).join("")}</select></label></p>
      <p><label>Cerrar al abrir el juego (separados por coma): <input type="text" data-gc="${i}" value="${esc((g.close || []).join(", "))}" placeholder="discord.exe, chrome.exe" style="width:260px"></label></p>
      <p class="note">Los programas se cierran sin guardar; nunca se cierran procesos de Windows. El plan de energía vuelve al anterior al cerrar el juego.</p></div><button class="btn danger" data-gx="${i}">Quitar</button></div>`).join("") || '<p class="note">Sin perfiles.</p>';
    m.querySelectorAll("[data-gp]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.gp].priority = c.checked; save(); });
    m.querySelectorAll("[data-gpl]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.gpl].powerPlan = c.value; save(); });
    m.querySelectorAll("[data-gc]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.gc].close = c.value.split(",").map(x => x.trim()).filter(Boolean); save(); });
    m.querySelectorAll("[data-gx]").forEach(c => c.onclick = async () => { cfg.games.splice(+c.dataset.gx, 1); await save(); renderGames(); });
  };
  renderGames();
  $("#addGame").onclick = async () => { const v = $("#newGame").value.trim(); if (!v) return; if (!cfg.games.some(g => g.exe.toLowerCase() === v.toLowerCase())) cfg.games.push({ exe: v, priority: false, powerPlan: "", close: [] }); await save(); renderGames(); $("#newGame").value = ""; };
  // estado en vivo
  const poll = async () => {
    if (view !== "overlay") return;
    try { const n = await api("overlay"); d.running = n.running; d.state = n.state; $("#ovStatus").innerHTML = ovStatus(d); $("#ovToggle").onclick = toggle.onclick; if (n.running) $("#ovPrev").innerHTML = ovPreview(cfg, n.state); } catch (e) { /* sigue */ }
    ovTimer = setTimeout(poll, 1500);
  };
  ovTimer = setTimeout(poll, 1500);
}

/* ---------- arranque ---------- */
setInterval(() => { api("heartbeat", {}).catch(() => { }); }, 3000);
api("heartbeat", {}).catch(() => { });
window.addEventListener("beforeunload", () => { /* el servidor se cierra solo al dejar de recibir latidos */ });
load().catch(e => { $("#main").innerHTML = `<div class="banner bad">No se pudo cargar: ${esc(e.message)}</div>`; });
