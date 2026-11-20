"use strict";
/* 3.2: tamaño de la interfaz, perfiles por juego en Modo juego, copias locales de la configuración y
   vista previa común para importar o restaurar (qué se activa y qué se desactiva). */

/* ---------- tamaño de la interfaz ---------- */
const UI_SCALES = [[90, "Chico (90%)"], [100, "Normal (100%)"], [110, "Grande (110%)"], [125, "Más grande (125%)"], [140, "Muy grande (140%)"]];

function applyUIScale(pct) {
  pct = +pct || 100;
  document.body.style.zoom = pct === 100 ? "" : String(pct / 100);
  try { localStorage.setItem("otScale", String(pct)); } catch (e) { /* sin almacenamiento */ }
}
(() => { let v = window.BOOT && window.BOOT.uiScale; if (!v) { try { v = +localStorage.getItem("otScale"); } catch (e) { /* nada */ } } if (v) applyUIScale(v); })();

/* ---------- vista previa de importar / restaurar ---------- */
// showImportPlan muestra qué va a pasar con una configuración (archivo importado o copia guardada) y la
// aplica si el usuario acepta. Con restore, ofrece también desactivar lo que hoy está aplicado y la copia
// no tiene, para volver exactamente a esa configuración.
async function showImportPlan(el, bundle, o = {}) {
  let plan;
  try { plan = await api("config/import", { bundle, apply: false }); } catch (err) { el.innerHTML = `<div class="banner bad">${esc(err.message)}</div>`; return; }
  const li = (a, why) => a.length ? `<ul>${a.map(x => `<li>${esc(x.name)}${why && x.reason ? ` <span class="note">— ${esc(x.reason)}</span>` : ""}</li>`).join("")}</ul>` : "";
  const rv = plan.toRevert || [];
  el.innerHTML = `<div class="row" style="display:block"><h4>Esto es lo que se va a hacer</h4>
    <p>Se guardarán los ajustes${plan.overlay ? " y la configuración del overlay" : ""} del archivo.</p>
    <p><b>Optimizaciones a activar (${plan.toApply.length})</b></p>${li(plan.toApply) || '<p class="note">Ninguna.</p>'}
    ${rv.length ? `<p><b>Aplicadas ahora que no están en el archivo (${rv.length})</b></p>${li(rv)}<label class="bar"><input type="checkbox" id="ipRev" ${o.restore ? "checked" : ""}> Desactivarlas también (volver exactamente a esta configuración)</label>` : ""}
    ${plan.already.length ? `<p><b>Ya aplicadas (${plan.already.length})</b></p>${li(plan.already)}` : ""}
    ${plan.skipped.length ? `<p><b>Se omiten (${plan.skipped.length})</b></p>${li(plan.skipped, true)}` : ""}
    <div class="btns" style="margin-top:10px"><button class="btn primary" id="ipGo">${o.restore ? "Restaurar" : "Importar"}</button><button class="btn" id="ipNo">Cancelar</button></div></div>`;
  $("#ipNo").onclick = () => { el.innerHTML = ""; };
  $("#ipGo").onclick = async () => {
    const rev = rv.length && $("#ipRev").checked ? rv : [];
    try {
      if ((plan.toApply.length || rev.length) && !S.admin) throw new Error("Para aplicar optimizaciones el programa necesita permisos de administrador.");
      if (plan.toApply.length || rev.length) await api("snapshots", { name: "Antes de restaurar «" + (bundle.name || "un archivo") + "»", auto: true });
      await api("config/import", { bundle, apply: true });
      S = await api("state");
      const nl = effectiveLang(S.settings && S.settings.lang);
      if (nl !== UILANG) { applyLang(nl); S = await api("state"); }
      if (typeof cacheIntroPrefs === "function") cacheIntroPrefs(introPrefs());
      el.innerHTML = "";
      if (rev.length) { render(); await runIds("Desactivar lo que no está en la copia", rev.map(x => x.id), "revert", true); }
      if (plan.toApply.length) { render(); await runIds(o.restore ? "Restaurar copia" : "Importar configuración", plan.toApply.map(x => x.id), "apply", true); }
      if (!rev.length && !plan.toApply.length) toast(o.restore ? "Copia restaurada." : "Configuración importada.");
      render();
    } catch (err) { toast(err.message, 6000); }
  };
}

/* ---------- copias locales (Configuración) ---------- */
function v32Settings(m) {
  const st = S.settings || {};
  m.insertAdjacentHTML("beforeend", `<h3 style="margin-top:22px">Apariencia</h3><div class="list">
    <div class="row"><div><h4>Tamaño de letra y botones</h4><p>Agranda o achica toda la ventana del programa.</p></div><select id="uiScale">${UI_SCALES.map(([v, n]) => `<option value="${v}" ${(st.uiScale || 100) === v ? "selected" : ""}>${n}</option>`).join("")}</select></div></div>
    <h3 style="margin-top:22px">Copias de tu configuración</h3>
    <p class="note">Fotos de cómo está todo (optimizaciones aplicadas, overlay, Modo juego y ajustes del programa), guardadas en una carpeta de tu PC. Antes de restaurar te mostramos qué se activa y qué se desactiva. Además se guarda una automáticamente antes de aplicar un perfil o el asistente, y antes de restaurar.</p>
    <div class="bar"><input type="text" id="snName" maxlength="60" placeholder="Nombre, por ejemplo Antes de probar Nagle" style="width:280px"><button class="btn primary" id="snSave">Guardar copia ahora</button><button class="btn" id="snDir">Abrir carpeta</button></div>
    <div id="snList" style="margin-top:10px"></div><div id="snOut"></div>`);
  v32Appearance(m);
  $("#uiScale").onchange = async e => {
    const v = +e.target.value;
    applyUIScale(v);
    try { S.settings = await api("settings", { uiScale: v }); } catch (err) { toast(err.message, 5000); }
  };
  const paint = async () => {
    let d;
    try { d = await api("snapshots"); } catch (e) { $("#snList").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
    const when = s => { const t = new Date(s); return isNaN(t) ? s : t.toLocaleString(UILANG === "en" ? "en-US" : "es-AR", { dateStyle: "short", timeStyle: "short" }); };
    $("#snList").innerHTML = d.list.length ? `<div class="list">${d.list.map(c => `<div class="row"><div><h4>${esc(c.name)}${c.auto ? ' <span class="tag">automática</span>' : ""}</h4><p>${when(c.at)} · ${c.tweaks} optimizaciones · versión ${esc(c.version)}</p></div>
      <span class="bar"><button class="btn" data-sr="${esc(c.id)}">Ver y restaurar</button><button class="btn" data-se="${esc(c.id)}">Exportar</button><button class="btn danger" data-sd="${esc(c.id)}">Borrar</button></span></div>`).join("")}</div>`
      : '<p class="note">Todavía no hay copias guardadas.</p>';
    $("#snList").querySelectorAll("[data-sr]").forEach(b => b.onclick = async () => {
      try { const bundle = await api("snapshots/get?id=" + encodeURIComponent(b.dataset.sr)); await showImportPlan($("#snOut"), bundle, { restore: true }); $("#snOut").scrollIntoView({ behavior: "smooth", block: "nearest" }); } catch (e) { toast(e.message, 5000); }
    });
    $("#snList").querySelectorAll("[data-se]").forEach(b => b.onclick = async () => {
      try {
        const bundle = await api("snapshots/get?id=" + encodeURIComponent(b.dataset.se));
        const a = document.createElement("a");
        a.href = URL.createObjectURL(new Blob([JSON.stringify(bundle, null, 1)], { type: "application/json" })); a.download = "overdrivetuner-" + b.dataset.se;
        document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 4000);
        toast("Copia exportada a Descargas.");
      } catch (e) { toast(e.message, 5000); }
    });
    $("#snList").querySelectorAll("[data-sd]").forEach(b => b.onclick = async () => {
      if (!await ask("Se borra esta copia de la carpeta. Tus ajustes actuales no cambian.", { title: "Borrar copia", ok: "Borrar", danger: true })) return;
      try { await api("snapshots/delete", { id: b.dataset.sd }); paint(); } catch (e) { toast(e.message, 5000); }
    });
  };
  $("#snSave").onclick = async () => {
    try { await api("snapshots", { name: $("#snName").value }); $("#snName").value = ""; toast("Copia guardada."); paint(); } catch (e) { toast(e.message, 5000); }
  };
  $("#snDir").onclick = () => api("snapshots/folder", {}).catch(e => toast(e.message, 5000));
  paint();
}

// autoSnapshot guarda una copia automática antes de un cambio grande (perfil, asistente). Si falla no frena nada.
async function autoSnapshot(name) { try { await api("snapshots", { name, auto: true }); } catch (e) { /* sigue */ } }

/* ---------- perfiles por juego (Modo juego) ---------- */
function gmProfiles(el, d, cfg, save) {
  if (!el) return;
  cfg.games = cfg.games || [];
  const plans = d.plans || [];
  el.innerHTML = `<h3 style="margin-top:22px">Perfiles por juego</h3>
    <p class="note">Lo que pasa al abrir cada juego, además de cerrar la lista de arriba. Todo vuelve a como estaba al cerrar el juego (menos los programas cerrados). Funcionan junto con el overlay, que es el que detecta el juego.</p>
    <div class="list" id="gpList"></div>
    <div class="gm-add"><input type="text" id="gpNew" placeholder="juego, ej. cs2.exe" style="width:200px" aria-label="Programa del juego"><button class="btn" id="gpAdd">Agregar juego</button><button class="btn" id="gpPick">Elegir de los programas abiertos</button></div><div id="gpPickOut"></div>`;
  const paint = () => {
    $("#gpList").innerHTML = cfg.games.length ? cfg.games.map((g, i) => `<div class="row"><div style="flex:1"><h4>${esc(g.exe)}</h4>
      <div class="gp-opts">
        <label><input type="checkbox" data-gk="priority" data-i="${i}" ${g.priority ? "checked" : ""}> Prioridad alta</label>
        <label><input type="checkbox" data-gk="quiet" data-i="${i}" ${g.quiet ? "checked" : ""}> Silenciar notificaciones</label>
        <label><input type="checkbox" data-gk="freeRam" data-i="${i}" ${g.freeRam ? "checked" : ""}> Liberar RAM al abrir</label>
        <label>Plan de energía: <select data-gpl="${i}"><option value="">no cambiar</option>${plans.map(p => `<option value="${esc(p.guid)}" ${g.powerPlan === p.guid ? "selected" : ""}>${esc(p.name)}</option>`).join("")}</select></label>
      </div>
      <p><label>Cerrar también al abrir este juego (separados por coma): <input type="text" data-gc="${i}" value="${esc((g.close || []).join(", "))}" placeholder="discord.exe, chrome.exe" style="width:260px"></label></p></div>
      <button class="btn danger" data-gx="${i}">Quitar</button></div>`).join("")
      : '<div class="row"><div><p>Todavía no hay perfiles. Agregá un juego, por ejemplo cs2.exe.</p></div></div>';
    el.querySelectorAll("[data-gk]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.i][c.dataset.gk] = c.checked; save(); });
    el.querySelectorAll("[data-gpl]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.gpl].powerPlan = c.value; save(); });
    el.querySelectorAll("[data-gc]").forEach(c => c.onchange = () => { cfg.games[+c.dataset.gc].close = c.value.split(",").map(x => gmName(x)).filter(Boolean); save(); });
    el.querySelectorAll("[data-gx]").forEach(c => c.onclick = async () => { cfg.games.splice(+c.dataset.gx, 1); await save(); paint(); });
  };
  const add = async raw => {
    const n = gmName(raw);
    if (!n) return;
    if (!/^[^\\/:*?"<>| ,;&]{1,60}\.exe$/i.test(n)) return toast("Poné solo el nombre del programa, por ejemplo chrome.exe", 5000);
    if (cfg.games.some(g => g.exe.toLowerCase() === n.toLowerCase())) return toast("Ya está en la lista.");
    cfg.games.push({ exe: n, priority: false, powerPlan: "", close: [], quiet: false, freeRam: false });
    await save(); paint();
  };
  $("#gpAdd").onclick = () => { add($("#gpNew").value); $("#gpNew").value = ""; };
  $("#gpPick").onclick = async () => {
    $("#gpPickOut").innerHTML = '<p class="note">Midiendo qué programas están abiertos (2 segundos)…</p>';
    try {
      const ps = (await api("procs")).filter(p => p.name && !GM_PROTECTED.includes(p.name.toLowerCase())).sort((a, b) => b.mb - a.mb);
      $("#gpPickOut").innerHTML = ps.length ? `<div class="btns" style="margin-top:8px">${ps.slice(0, 30).map(p => `<button class="btn" data-pk="${esc(p.name)}.exe">${esc(p.name)}.exe · ${p.mb} MB</button>`).join("")}</div>` : '<p class="note">No encontré programas abiertos.</p>';
      $("#gpPickOut").querySelectorAll("[data-pk]").forEach(b => b.onclick = () => { add(b.dataset.pk); $("#gpPickOut").innerHTML = ""; });
    } catch (e) { $("#gpPickOut").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; }
  };
  paint();
}

/* ---------- tema claro / oscuro ---------- */
const THEMES = [["", "Oscuro"], ["light", "Claro"], ["system", "Igual que Windows"]];
const lightQuery = window.matchMedia ? matchMedia("(prefers-color-scheme: light)") : null;

function applyTheme(t) {
  const light = t === "light" || (t === "system" && lightQuery && lightQuery.matches);
  if (light) document.documentElement.dataset.theme = "light"; else delete document.documentElement.dataset.theme;
  try { localStorage.setItem("otTheme", t || ""); } catch (e) { /* sin almacenamiento */ }
}
(() => {
  let t = window.BOOT && window.BOOT.theme;
  if (t === undefined) { try { t = localStorage.getItem("otTheme") || ""; } catch (e) { t = ""; } }
  applyTheme(t || "");
  if (lightQuery) lightQuery.addEventListener("change", () => { const cur = (S && S.settings && S.settings.theme) || ""; if (cur === "system") applyTheme(cur); });
})();

function v32Appearance(m) {
  const st = S.settings || {};
  const row = m.querySelector("#uiScale").closest(".row");
  row.insertAdjacentHTML("beforebegin", `<div class="row"><div><h4>Tema</h4><p>Oscuro (el de siempre), claro o el mismo que usa Windows.</p></div><select id="uiTheme">${THEMES.map(([v, n]) => `<option value="${v}" ${(st.theme || "") === v ? "selected" : ""}>${n}</option>`).join("")}</select></div>`);
  $("#uiTheme").onchange = async e => {
    applyTheme(e.target.value);
    try { S.settings = await api("settings", { theme: e.target.value }); } catch (err) { toast(err.message, 5000); }
  };
}

/* ---------- Inicio personalizable ---------- */
const HOME_BLOCKS = [["hero", "Puntaje y perfiles"], ["status", "Estado de tu PC"], ["quick", "Accesos rápidos"], ["hardware", "Tu equipo"], ["recent", "Últimos cambios"]];
const HOME_DEFAULT = ["hero", "quick", "hardware"];

function homeLayout() { const h = S.settings && S.settings.home; return h && h.length ? h : HOME_DEFAULT; }

function homeBlocks(blocks) {
  const days = iso => { const t = Date.parse(iso); return isNaN(t) || t < Date.UTC(2001, 0, 1) ? -1 : Math.floor((Date.now() - t) / 864e5); };
  const up = S.uptimeH || 0, clean = days(S.settings && S.settings.lastClean);
  const applied = S.tweaks.filter(t => t.applied).length, drift = S.tweaks.filter(t => t.drift).length;
  blocks.status = `<h3 style="margin-top:22px">Estado de tu PC</h3><div class="grid">
    ${hwCard("Encendida", up >= 48 ? `hace ${Math.floor(up / 24)} días` : up >= 1 ? `hace ${up} horas` : "hace menos de una hora", up >= 168 ? "Conviene reiniciar" : "")}
    ${hwCard("Optimizaciones", String(applied), drift ? `${drift} deshechas por Windows` : "todas en su lugar")}
    ${hwCard("Última limpieza", clean < 0 ? "sin registro" : clean === 0 ? "hoy" : `hace ${clean} días`, "")}</div>`;
  blocks.quick = `<h3 style="margin-top:22px">Accesos rápidos</h3><div class="btns" id="homeQuick">
    <button class="btn" data-q="ram" ${S.admin ? "" : "disabled"}>Liberar RAM</button><button class="btn" data-q="clean">Limpieza</button><button class="btn" data-q="fluidez">Diagnóstico</button>
    <button class="btn" data-q="bench">Benchmark</button><button class="btn" data-q="overlay">Overlay</button><button class="btn" data-q="snap">Guardar copia de la configuración</button></div>`;
  blocks.recent = `<h3 style="margin-top:22px">Últimos cambios</h3><div id="homeRecent"><p class="note">Cargando…</p></div>`;
  return homeLayout().map(id => blocks[id] ? `<div class="home-b">${blocks[id]}</div>` : "").join("");
}

function wireHomeExtras(m) {
  m.querySelectorAll("[data-q]").forEach(b => b.onclick = async () => {
    const q = b.dataset.q;
    if (q === "ram") {
      b.disabled = true;
      try { const r = await api("memory/free", {}); toast(`Memoria libre: ${(r.beforeMB / 1024).toFixed(1)} → ${(r.afterMB / 1024).toFixed(1)} GB`, 5000); } catch (e) { toast(e.message, 5000); }
      b.disabled = false;
    } else if (q === "snap") {
      try { await api("snapshots", { name: "" }); toast("Copia guardada."); } catch (e) { toast(e.message, 5000); }
    } else { view = q; render(); $("#main").scrollTop = 0; }
  });
  const rec = m.querySelector("#homeRecent");
  if (rec) api("history").then(list => {
    const l = (list || []).slice(0, 5);
    rec.innerHTML = l.length ? `<div class="list">${l.map(h => `<div class="row"><div><h4><span class="tag">${h.action === "apply" ? "Aplicado" : h.action === "reapply" ? "Vuelto a aplicar" : "Revertido"}</span> ${esc((S.tweaks.find(t => t.id === h.id) || {}).name || h.name)}</h4></div></div>`).join("")}</div>`
      : '<p class="note">Todavía no hay cambios registrados.</p>';
  }).catch(() => { rec.innerHTML = ""; });
}

function editHome() {
  let order = homeLayout().slice();
  const dlg = v31Dialog("");
  const paint = () => {
    const all = order.concat(HOME_BLOCKS.map(b => b[0]).filter(id => !order.includes(id)));
    dlg.el.innerHTML = `<h3>Personalizar Inicio</h3><p class="note">Elegí qué bloques ver y en qué orden.</p>
      <div class="list">${all.map(id => { const on = order.includes(id), i = order.indexOf(id); return `<div class="row"><div><label class="bar"><input type="checkbox" data-hv="${id}" ${on ? "checked" : ""}> ${HOME_BLOCKS.find(b => b[0] === id)[1]}</label></div>
        <span class="bar"><button class="btn" data-hu="${id}" ${on && i > 0 ? "" : "disabled"}>Subir</button><button class="btn" data-hd="${id}" ${on && i < order.length - 1 ? "" : "disabled"}>Bajar</button></span></div>`; }).join("")}</div>
      <div class="end"><button class="btn" data-reset>Volver al de siempre</button><button class="btn" data-c>Cancelar</button><button class="btn primary" data-s>Guardar</button></div>`;
    dlg.el.querySelectorAll("[data-hv]").forEach(c => c.onchange = () => { order = c.checked ? order.concat(c.dataset.hv) : order.filter(x => x !== c.dataset.hv); paint(); });
    const mv = (id, d) => { const i = order.indexOf(id); [order[i], order[i + d]] = [order[i + d], order[i]]; paint(); };
    dlg.el.querySelectorAll("[data-hu]").forEach(b => b.onclick = () => mv(b.dataset.hu, -1));
    dlg.el.querySelectorAll("[data-hd]").forEach(b => b.onclick = () => mv(b.dataset.hd, 1));
    dlg.el.querySelector("[data-reset]").onclick = () => { order = HOME_DEFAULT.slice(); paint(); };
    dlg.el.querySelector("[data-c]").onclick = dlg.close;
    dlg.el.querySelector("[data-s]").onclick = async () => {
      if (!order.length) return toast("Dejá al menos un bloque.");
      try { S.settings = await api("settings", { home: order }); dlg.close(); render(); } catch (e) { toast(e.message, 5000); }
    };
  };
  paint();
}

/* ---------- widget de escritorio (sección Overlay) ---------- */
function ovWidget(el, cfg, save) {
  if (!el) return;
  const corners = [["tl", "Arriba izquierda"], ["tr", "Arriba derecha"], ["bl", "Abajo izquierda"], ["br", "Abajo derecha"]];
  el.innerHTML = `<h3 style="margin-top:22px">Widget de escritorio</h3><div class="list">
    <div class="row"><div><h4>Mostrar CPU, GPU, RAM y temperaturas en el escritorio</h4><p>Una tarjeta detrás de las ventanas, con el mismo estilo que el overlay. Se esconde mientras jugás (ahí está el overlay) y funciona mientras el overlay está abierto, por ejemplo si arranca con Windows.</p></div><button class="sw" id="wgOn" role="switch" aria-checked="${cfg.widget ? "true" : "false"}" aria-label="Widget de escritorio"></button></div>
    <div class="row"><div><h4>Esquina del widget</h4></div><select id="wgPos">${corners.map(([v, n]) => `<option value="${v}" ${(cfg.widgetPos || "tr") === v ? "selected" : ""}>${n}</option>`).join("")}</select></div></div>`;
  $("#wgOn").onclick = () => { cfg.widget = !cfg.widget; $("#wgOn").setAttribute("aria-checked", cfg.widget); save(); };
  $("#wgPos").onchange = e => { cfg.widgetPos = e.target.value; save(); };
}
