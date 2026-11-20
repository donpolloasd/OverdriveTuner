"use strict";
/* 3.2: «Espacio en disco» (analizador al estilo WizTree, con duplicados) y, en Limpieza, la Papelera y los
   puntos de restauración viejos. Borrar desde el analizador siempre manda a la Papelera. */

let spacePath = "", spaceTab = "map", spaceTimer = 0;

const gb = b => b >= 1 << 30 ? (b / (1 << 30)).toFixed(1) + " GB" : b >= 1 << 20 ? (b / (1 << 20)).toFixed(0) + " MB" : Math.max(0, Math.round(b / 1024)) + " KB";
const joinPath = (a, b) => a.endsWith("\\") ? a + b : a + "\\" + b;

// treemap reparte un rectángulo entre los elementos según su tamaño (algoritmo «squarified»: rectángulos
// lo más cuadrados posible, como en WizTree).
function treemap(items, x, y, w, h) {
  const total = items.reduce((a, b) => a + b.size, 0);
  if (!total || w <= 0 || h <= 0) return [];
  const k = w * h / total, out = [];
  let rect = { x, y, w, h }, row = [];
  const worst = (r, side) => { const s = r.reduce((a, b) => a + b.a, 0); let mx = 0, mn = Infinity; r.forEach(n => { mx = Math.max(mx, n.a); mn = Math.min(mn, n.a); }); return Math.max(side * side * mx / (s * s), s * s / (side * side * mn)); };
  const lay = r => {
    const s = r.reduce((a, b) => a + b.a, 0);
    if (rect.w >= rect.h) { const cw = s / rect.h; let cy = rect.y; r.forEach(n => { const ch = n.a / cw; out.push({ it: n.it, x: rect.x, y: cy, w: cw, h: ch }); cy += ch; }); rect = { x: rect.x + cw, y: rect.y, w: rect.w - cw, h: rect.h }; }
    else { const ch = s / rect.w; let cx = rect.x; r.forEach(n => { const cw = n.a / ch; out.push({ it: n.it, x: cx, y: rect.y, w: cw, h: ch }); cx += cw; }); rect = { x: rect.x, y: rect.y + ch, w: rect.w, h: rect.h - ch }; }
  };
  for (const n of items.filter(i => i.size > 0).map(it => ({ it, a: it.size * k }))) {
    const side = Math.min(rect.w, rect.h);
    if (!row.length || worst(row.concat(n), side) <= worst(row, side)) row.push(n); else { lay(row); row = [n]; }
  }
  if (row.length) lay(row);
  return out;
}

async function spaceRecycle(path, size, after) {
  if (!await ask(`Se va a mandar a la Papelera «${path}»${size ? " (" + gb(size) + ")" : ""}.\nLo podés recuperar desde la Papelera mientras no la vacíes.`, { title: "Mandar a la Papelera", ok: "Mandar a la Papelera", danger: true })) return;
  try { await api("space/recycle", { path, confirm: 1 }); toast("Enviado a la Papelera."); if (after) after(); } catch (e) { toast(e.message, 7000); }
}
const spaceOpen = path => api("space/open", { path }).catch(e => toast(e.message, 5000));

async function renderSpace(m) {
  clearTimeout(spaceTimer);
  const head = `<div class="top"><div><h2>Espacio en disco</h2><p>Qué ocupa lugar en tus discos, como WizTree: un mapa por carpetas, los archivos más grandes, el espacio por tipo y los archivos repetidos. Borrar siempre manda a la Papelera, y las carpetas de Windows y de los programas instalados están protegidas.</p></div></div>`;
  let st;
  try { st = await api("space"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "space") return;
  const vols = st.volumes || [];
  const pick = `<div class="bar"><select id="spRoot">${vols.map(v => `<option value="${esc(v.root)}" ${(st.root || "C:\\") === v.root ? "selected" : ""}>${esc(v.root)} · ${gb(v.free)} libres de ${gb(v.total)}</option>`).join("")}</select>
    ${st.running ? '<button class="btn" id="spCancel">Cancelar</button>' : '<button class="btn primary" id="spGo">Analizar</button>'}</div>`;
  if (st.running) {
    m.innerHTML = head + banners() + pick + `<p class="note" style="margin-top:12px">Analizando ${esc(st.root)}… ${Number(st.files || 0).toLocaleString()} archivos · ${gb(st.bytes || 0)} · ${Math.round((st.elapsedMs || 0) / 1000)} s</p>`;
    $("#spCancel").onclick = () => api("space/scan", { cancel: true }).catch(() => { });
    spaceTimer = setTimeout(() => { if (view === "space") renderSpace(m); }, 800);
    return;
  }
  m.innerHTML = head + banners() + pick + (st.error ? `<div class="banner bad" style="margin-top:12px">${esc(st.error)}</div>` : "") + '<div id="spBody"></div>';
  $("#spGo").onclick = async () => { try { spacePath = ""; await api("space/scan", { root: $("#spRoot").value }); renderSpace(m); } catch (e) { toast(e.message, 5000); } };
  if (!st.done) { $("#spBody").innerHTML = '<p class="note" style="margin-top:12px">Elegí una unidad y tocá Analizar. En un disco lleno tarda alrededor de medio minuto.</p>'; return; }
  const d = st.done;
  if (!spacePath || !spacePath.toLowerCase().startsWith(d.root.toLowerCase())) spacePath = d.root;
  const tabs = [["map", "Mapa"], ["files", "Archivos más grandes"], ["types", "Por tipo"], ["dups", "Duplicados"]];
  $("#spBody").innerHTML = `<div class="banner" style="margin-top:12px">${esc(d.root)}: ${gb(d.bytes)} en ${Number(d.files).toLocaleString()} archivos y ${Number(d.dirs).toLocaleString()} carpetas · analizado en ${(d.tookMs / 1000).toFixed(1)} s${d.skipped ? ` · ${d.skipped} carpetas sin permiso de lectura` : ""}</div>
    <div class="btns sp-tabs">${tabs.map(([id, n]) => `<button class="btn ${spaceTab === id ? "primary" : ""}" data-tab="${id}">${n}</button>`).join("")}</div><div id="spTab"></div>`;
  m.querySelectorAll("[data-tab]").forEach(b => b.onclick = () => { spaceTab = b.dataset.tab; renderSpace(m); });
  const el = $("#spTab");
  if (spaceTab === "map") return spaceMap(el, m);
  if (spaceTab === "dups") return spaceDups(el, m, false);
  let top;
  try { top = await api("space/top"); } catch (e) { el.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (spaceTab === "files") {
    el.innerHTML = `<div class="list">${top.files.map((f, i) => `<div class="row"><div style="min-width:0"><h4>${esc(f.path.split("\\").pop())} <span class="tag">${gb(f.size)}</span></h4><p class="note sp-path">${esc(f.path)}</p></div>
      <span class="bar"><button class="btn" data-op="${i}">Explorador</button>${f.protected ? "" : `<button class="btn danger" data-rc="${i}">Papelera</button>`}</span></div>`).join("")}</div>`;
    el.querySelectorAll("[data-op]").forEach(b => b.onclick = () => spaceOpen(top.files[+b.dataset.op].path));
    el.querySelectorAll("[data-rc]").forEach(b => b.onclick = () => { const f = top.files[+b.dataset.rc]; spaceRecycle(f.path, f.size, () => renderSpace(m)); });
  } else {
    const max = Math.max(1, ...top.types.map(t => t.bytes));
    el.innerHTML = `<div class="list">${top.types.map(t => `<div class="row"><div style="flex:1"><h4>${esc(t.ext)} <span class="note">${Number(t.count).toLocaleString()} archivos</span></h4><div class="sp-bar"><i style="width:${(t.bytes / max * 100).toFixed(1)}%"></i></div></div><b>${gb(t.bytes)}</b></div>`).join("")}</div>`;
  }
}

async function spaceMap(el, m) {
  let d;
  try { d = await api("space/dir?path=" + encodeURIComponent(spacePath)); } catch (e) { spacePath = ""; el.innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
  const v = d.view, root = d.root;
  // Migas de pan: de la raíz analizada a la carpeta actual.
  const rel = v.path.length > root.length ? v.path.slice(root.length).replace(/^\\/, "").split("\\") : [];
  const crumbs = [[root, root]].concat(rel.map((p, i) => [p, joinPath(root, rel.slice(0, i + 1).join("\\"))]));
  const items = v.kids.map(k => ({ name: k.name, size: k.size, files: k.files, dir: true }));
  if (v.loose > 0) items.push({ name: "Archivos en esta carpeta", size: v.loose, loose: true });
  if (v.others > 0) items.push({ name: "Otras carpetas", size: v.others, loose: true });
  items.sort((a, b) => b.size - a.size);
  el.innerHTML = `<div class="sp-crumbs">${crumbs.map(([n, p], i) => `${i ? '<span class="note"> › </span>' : ""}<a href="#" data-cd="${esc(p)}">${esc(n)}</a>`).join("")} <span class="note">· ${gb(v.size)}</span></div>
    <div class="sp-map" id="spMap"></div>
    <div class="list" style="margin-top:12px">${items.slice(0, 60).map((it, i) => `<div class="row"><div style="flex:1;min-width:0"><h4>${it.dir ? `<a href="#" data-cd="${esc(joinPath(v.path, it.name))}">${esc(it.name)}</a>` : esc(it.name)}</h4>
      <div class="sp-bar"><i style="width:${v.size ? (it.size / v.size * 100).toFixed(1) : 0}%"></i></div><p class="note">${gb(it.size)} · ${v.size ? (it.size / v.size * 100).toFixed(1) : 0}%${it.files ? ` · ${Number(it.files).toLocaleString()} archivos` : ""}</p></div>
      ${it.dir ? `<span class="bar"><button class="btn" data-op="${i}">Explorador</button><button class="btn danger" data-rc="${i}">Papelera</button></span>` : ""}</div>`).join("")}</div>`;
  const box = $("#spMap"), W = box.clientWidth, H = box.clientHeight;
  box.innerHTML = treemap(items, 0, 0, W, H).map((r, i) => {
    const big = r.w > 70 && r.h > 34;
    return `<div class="sp-cell ${r.it.dir ? "dir" : "loose"}" style="left:${r.x}px;top:${r.y}px;width:${Math.max(0, r.w - 2)}px;height:${Math.max(0, r.h - 2)}px;--h:${(i * 47) % 360}" title="${esc(r.it.name)} · ${gb(r.it.size)}" ${r.it.dir ? `data-cd="${esc(joinPath(v.path, r.it.name))}"` : ""}>${big ? `<b>${esc(r.it.name)}</b><small>${gb(r.it.size)}</small>` : ""}</div>`;
  }).join("");
  el.querySelectorAll("[data-cd]").forEach(a => a.onclick = e => { e.preventDefault(); spacePath = a.dataset.cd; spaceMap(el, m); });
  el.querySelectorAll("[data-op]").forEach(b => b.onclick = () => spaceOpen(joinPath(v.path, items[+b.dataset.op].name)));
  el.querySelectorAll("[data-rc]").forEach(b => b.onclick = () => { const it = items[+b.dataset.rc]; spaceRecycle(joinPath(v.path, it.name), it.size, () => spaceMap(el, m)); });
}

async function spaceDups(el, m, run) {
  el.innerHTML = `<p class="note">Busca archivos de 1 MB o más con el mismo contenido: primero compara el tamaño, después el comienzo y al final el archivo entero. Los archivos que están solo en la nube (OneDrive) no se leen. Dejá siempre al menos una copia.</p>
    <div class="bar"><button class="btn primary" id="dpGo">${run ? "Buscando…" : "Buscar archivos repetidos (hasta 1 minuto)"}</button></div><div id="dpOut"></div>`;
  const go = async post => {
    $("#dpGo").disabled = true; $("#dpGo").textContent = "Buscando…";
    let d;
    try { d = post ? await api("space/dups", {}) : await api("space/dups"); } catch (e) { $("#dpOut").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
    $("#dpGo").disabled = false; $("#dpGo").textContent = "Buscar de nuevo";
    const groups = d.groups.filter(g => g.files.length > 1);
    const wasted = groups.reduce((a, g) => a + g.size * (g.files.length - 1), 0);
    $("#dpOut").innerHTML = groups.length ? `<div class="banner" style="margin-top:12px">${groups.length} grupos de archivos repetidos · ${gb(wasted)} que se liberarían dejando una copia de cada uno${d.complete ? "" : " · búsqueda parcial: se cortó al minuto, empezando por los más grandes"}</div>
      ${groups.map((g, gi) => `<div class="row" style="display:block"><h4>${gb(g.size)} cada uno · ${g.files.length} copias</h4>${g.files.map((f, fi) => `<div class="bar sp-dup"><span class="sp-path" style="flex:1">${esc(f)}</span><button class="btn" data-dop="${gi}:${fi}">Explorador</button><button class="btn danger" data-drc="${gi}:${fi}">Papelera</button></div>`).join("")}</div>`).join("")}`
      : `<p class="note" style="margin-top:12px">No se encontraron archivos repetidos de 1 MB o más${d.complete ? "" : " en el minuto de búsqueda"}.</p>`;
    const at = s => { const [a, b] = s.split(":").map(Number); return [groups[a], groups[a].files[b]]; };
    $("#dpOut").querySelectorAll("[data-dop]").forEach(b => b.onclick = () => spaceOpen(at(b.dataset.dop)[1]));
    $("#dpOut").querySelectorAll("[data-drc]").forEach(b => b.onclick = () => {
      const [g, f] = at(b.dataset.drc);
      if (g.files.length < 2) return toast("Es la última copia: no se borra desde acá.", 5000);
      spaceRecycle(f, g.size, () => go(false));
    });
  };
  $("#dpGo").onclick = () => go(true);
  try { const d = await api("space/dups"); if (d.groups && d.groups.length) go(false); } catch (e) { /* todavía no se buscó */ }
}

/* ---------- Limpieza: Papelera y puntos de restauración ---------- */
function v33Clean(el) {
  if (!el) return;
  el.innerHTML = `<h3 style="margin-top:22px">Papelera</h3><div class="list"><div class="row"><div><h4 id="rbInfo">Papelera</h4><p>Lo que borraste sigue ocupando lugar hasta que vacíes la Papelera.</p></div><button class="btn danger" id="rbGo">Vaciar la Papelera</button></div></div>
    <h3 style="margin-top:22px">Puntos de restauración</h3><p class="note">Windows guarda puntos para volver atrás si algo sale mal. Los viejos pueden ocupar varios GB.</p>
    <div class="bar"><button class="btn" id="rpGo" ${S.admin ? "" : "disabled"}>Ver puntos de restauración</button></div><div id="rpOut"></div>`;
  let rb = { bytes: 0, items: 0 };
  const loadRb = async () => { try { rb = await api("recyclebin"); $("#rbInfo").textContent = rb.items ? `Papelera: ${gb(rb.bytes)} en ${rb.items} elementos` : "La Papelera está vacía"; $("#rbGo").disabled = !rb.items; } catch (e) { $("#rbInfo").textContent = e.message; } };
  loadRb();
  $("#rbGo").onclick = async () => {
    if (!await ask(`Se borra definitivamente todo lo que hay en la Papelera (${gb(rb.bytes)}, ${rb.items} elementos).\nNo se puede deshacer.`, { title: "Vaciar la Papelera", ok: "Vaciar", danger: true })) return;
    try { await api("recyclebin", { confirm: 1 }); toast("Papelera vaciada."); loadRb(); } catch (e) { toast(e.message, 6000); }
  };
  $("#rpGo").onclick = async () => {
    $("#rpOut").innerHTML = '<p class="note">Leyendo…</p>';
    let d;
    try { d = await api("restorepoints"); } catch (e) { $("#rpOut").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; return; }
    const pts = d.points.slice().sort((a, b) => b.seq - a.seq);
    $("#rpOut").innerHTML = `<div class="banner" style="margin-top:12px">${pts.length} puntos de restauración · ocupan ${gb(d.used)}</div>
      ${pts.length ? `<div class="list">${pts.map((p, i) => `<div class="row"><div><h4>${esc(p.desc)}${i === 0 ? ' <span class="tag ok">el más reciente</span>' : ""}</h4><p class="note">${esc(p.at)}</p></div></div>`).join("")}</div>` : ""}
      ${d.shadows > 1 ? `<div class="bar" style="margin-top:10px"><button class="btn danger" id="rpDel" ${S.admin ? "" : "disabled"}>Borrar todos menos el más reciente</button></div>` : ""}`;
    const del = $("#rpDel");
    if (del) del.onclick = async () => {
      if (!await ask("Se borran todos los puntos de restauración del disco del sistema menos el más reciente.\nSi algo sale mal, solo vas a poder volver a ese último punto. También se borran otras instantáneas del disco del sistema (por ejemplo, las de programas de copia de seguridad).", { title: "Borrar puntos de restauración viejos", ok: "Continuar", danger: true })) return;
      if (!await ask("¿Seguro? Esto no se puede deshacer.", { title: "Confirmar otra vez", ok: "Borrar", danger: true })) return;
      try { const r = await api("restorepoints", { confirm: 2 }); toast(`Se borraron ${r.deleted} puntos de restauración.`, 5000); $("#rpGo").click(); } catch (e) { toast(e.message, 6000); }
    };
  };
}
