"use strict";
/* 3.1: ajustes que Windows deshizo, asistente por género, novedades al actualizar, «Después de formatear»,
   reportar un problema, overlay como imagen, programas y juegos sin usar, drivers viejos, latencia DPC,
   throttling, notificaciones silenciadas al jugar y recordatorios. */

/* ---------- ajustes que Windows deshizo (Inicio) ---------- */
function driftList() { return (S.tweaks || []).filter(t => t.drift); }

function driftBanner() {
  const l = driftList();
  if (!l.length) return "";
  const wu = S.winUpdate;
  const why = wu ? `Windows se actualizó (compilación ${wu.from} → ${wu.to}) y deshizo ${l.length} ajuste(s).` : `Windows (u otro programa) deshizo ${l.length} ajuste(s) que habías aplicado.`;
  return `<div class="banner warn" id="driftBox"><div><b>${why}</b></div><div class="note">${l.slice(0, 5).map(t => esc(t.name)).join(" · ")}${l.length > 5 ? " · …" : ""}</div>
    <div class="bar" style="margin-top:8px"><button class="btn primary" id="driftFix" ${S.admin ? "" : "disabled"}>Volver a aplicarlos</button></div></div>`;
}

function wireDrift() {
  const b = $("#driftFix");
  if (!b) return;
  b.onclick = async () => {
    const l = driftList();
    modal.open("Volver a aplicar ajustes");
    try {
      const res = await api("reapply", { ids: l.map(t => t.id) });
      res.forEach(r => { const t = l.find(x => x.id === r.id) || { name: r.id }; modal.log((r.ok ? "✔ " : "✖ ") + t.name + (r.ok ? "" : ": " + (r.errors || []).join("; ")), r.ok ? "ok" : "bad"); });
      if (res.some(r => r.reboot)) modal.log("Algunos cambios necesitan reiniciar la PC.", "note");
    } catch (e) { modal.log(e.message, "bad"); }
    modal.progress(1); modal.done();
    await refreshTweaks();
  };
}

/* ---------- diálogo propio con contenido HTML (asistente, novedades, reporte) ---------- */
function v31Dialog(html, o = {}) {
  const v = document.createElement("div");
  v.className = "veil show"; v.style.zIndex = 80;
  v.innerHTML = `<div class="modal dlg wide" role="dialog" aria-modal="true">${html}</div>`;
  const close = () => { document.removeEventListener("keydown", key, true); v.remove(); if (o.onClose) o.onClose(); };
  const key = e => { if (e.key === "Escape" && !o.sticky) { e.stopPropagation(); close(); } };
  document.addEventListener("keydown", key, true);
  document.body.appendChild(v);
  return { el: v.firstElementChild, close };
}

/* ---------- novedades al actualizar ---------- */
async function whatsNew() {
  let d;
  try { d = await api("whatsnew?lang=" + (window.UILANG || "es")); } catch (e) { return; }
  if (!d.releases || !d.releases.length) return;
  const dlg = v31Dialog(`<h3>Novedades de la versión ${esc(d.version)}</h3>
    ${d.releases.map(r => `<h4 style="margin:14px 0 4px">${esc(r.version)} · ${esc(r.title)}</h4><ul>${r.changes.map(c => `<li><b>${esc(c.t)}.</b> ${esc(c.x)}</li>`).join("")}</ul>`).join("")}
    <div class="end"><button class="btn primary" data-ok>Entendido</button></div>`, { onClose: () => api("whatsnew", {}).catch(() => { }) });
  dlg.el.querySelector("[data-ok]").onclick = dlg.close;
}

/* ---------- asistente de primera vez (por género) ---------- */
const WZ_GENRES = [["shooter", "Shooters / FPS"], ["comp", "Competitivos con anti-cheat"], ["sim", "Simuladores"], ["sports", "Deportes"], ["aaa", "Mundo abierto / AAA"], ["strategy", "Estrategia / MMO"], ["casual", "Casual / indie"]];

// wizardPick elige los ajustes según las respuestas. Solo ajustes seguros: nunca anti-cheat ni avanzados.
function wizardPick(a) {
  const base = S.tweaks.filter(t => t.suitable && !t.ac && t.risk < 2 && t.cat !== "apps" && t.cat !== "advanced" && !t.active);
  const g = new Set(a.genres);
  const onlyCasual = g.size === 1 && g.has("casual");
  let pick = base.filter(t => t.rec && (!onlyCasual || t.risk === 0));
  if ((g.has("shooter") || g.has("comp") || g.has("sports")) && a.goal === "fps") pick = pick.concat(base.filter(t => t.comp && !pick.includes(t)));
  if (g.has("sim") || g.has("aaa") || g.has("strategy")) pick = pick.concat(base.filter(t => t.cat === "memory" && t.risk === 0 && !pick.includes(t)));
  if (a.goal === "balance") pick = pick.filter(t => !t.comp || t.rec);
  if (a.laptop) pick = pick.filter(t => !(t.cat === "power" && t.risk > 0));
  return pick;
}

function showWizard(onDone) {
  const a = { genres: [], laptop: !!(S.hardware && S.hardware.laptop), goal: "fps" };
  const done = () => { api("settings", { wizardDone: true }).catch(() => { }); window.BOOT = Object.assign(window.BOOT || {}, { wizardDone: true }); if (onDone) onDone(); };
  const dlg = v31Dialog("", { sticky: true });
  const step = n => {
    const el = dlg.el;
    if (n === 1) {
      el.innerHTML = `<h3>¿Qué tipo de juegos jugás?</h3><p class="note">Podés elegir varios. Con esto armamos una recomendación; antes de aplicar nada te mostramos la lista.</p>
        <div class="wz-opts">${WZ_GENRES.map(([id, n]) => `<button class="btn wz ${a.genres.includes(id) ? "on" : ""}" data-g="${id}" aria-pressed="${a.genres.includes(id)}">${n}</button>`).join("")}</div>
        <div class="end"><button class="btn" data-skip>Saltear</button><button class="btn primary" data-next ${a.genres.length ? "" : "disabled"}>Siguiente</button></div>`;
      el.querySelectorAll("[data-g]").forEach(b => b.onclick = () => { const g = b.dataset.g; a.genres = a.genres.includes(g) ? a.genres.filter(x => x !== g) : a.genres.concat(g); step(1); });
      el.querySelector("[data-next]").onclick = () => step(2);
    } else if (n === 2) {
      el.innerHTML = `<h3>¿Tu equipo es una PC de escritorio o una notebook?</h3><p class="note">En notebook no tocamos el plan de energía de máximo rendimiento: gasta batería y calienta.</p>
        <div class="wz-opts"><button class="btn wz ${!a.laptop ? "on" : ""}" data-l="0">PC de escritorio</button><button class="btn wz ${a.laptop ? "on" : ""}" data-l="1">Notebook</button></div>
        <div class="end"><button class="btn" data-back>Atrás</button><button class="btn primary" data-next>Siguiente</button></div>`;
      el.querySelectorAll("[data-l]").forEach(b => b.onclick = () => { a.laptop = b.dataset.l === "1"; step(2); });
      el.querySelector("[data-back]").onclick = () => step(1);
      el.querySelector("[data-next]").onclick = () => step(3);
    } else if (n === 3) {
      el.innerHTML = `<h3>¿Qué preferís?</h3>
        <div class="wz-opts"><button class="btn wz ${a.goal === "fps" ? "on" : ""}" data-o="fps">Máximos FPS y menor latencia</button><button class="btn wz ${a.goal === "balance" ? "on" : ""}" data-o="balance">Equilibrio: rendimiento, temperatura y consumo</button></div>
        <div class="end"><button class="btn" data-back>Atrás</button><button class="btn primary" data-next>Ver recomendación</button></div>`;
      el.querySelectorAll("[data-o]").forEach(b => b.onclick = () => { a.goal = b.dataset.o; step(3); });
      el.querySelector("[data-back]").onclick = () => step(2);
      el.querySelector("[data-next]").onclick = () => step(4);
    } else {
      const pick = wizardPick(a);
      el.innerHTML = `<h3>Tu recomendación</h3>${pick.length
        ? `<p>Vamos a aplicar ${pick.length} ajuste(s) seguros para tu equipo. Antes se crea un punto de restauración y cada uno se puede revertir.</p><ul class="wz-list">${pick.map(t => `<li>${esc(t.name)}</li>`).join("")}</ul>`
        : `<p>Tu PC ya tiene aplicados los ajustes que te recomendaríamos.</p>`}
        ${S.admin ? "" : '<div class="banner bad">El programa no tiene permisos de administrador, así que no puede aplicar cambios.</div>'}
        <div class="end"><button class="btn" data-back>Atrás</button><button class="btn" data-skip>Ahora no</button>${pick.length ? `<button class="btn primary" data-go ${S.admin ? "" : "disabled"}>Aplicar</button>` : ""}</div>`;
      el.querySelector("[data-back]").onclick = () => step(3);
      const go = el.querySelector("[data-go]");
      if (go) go.onclick = async () => {
        dlg.close(); done();
        modal.open("Aplicando la recomendación");
        modal.log("Creando punto de restauración…", "note");
        try { await api("restorepoint", {}); modal.log("Punto de restauración creado.", "ok"); } catch (e) { modal.log(e.message, "bad"); }
        for (let i = 0; i < pick.length; i++) {
          const t = pick[i];
          try { const r = await api("apply", { id: t.id, confirm: 1 }); modal.log((r.ok ? "✔ " : "✖ ") + t.name, r.ok ? "ok" : "bad"); } catch (e) { modal.log("✖ " + t.name + ": " + e.message, "bad"); }
          modal.progress((i + 1) / pick.length);
        }
        modal.done();
        await refreshTweaks();
      };
    }
    const sk = el.querySelector("[data-skip]");
    if (sk) sk.onclick = () => { dlg.close(); done(); };
  };
  step(1);
}

// Instalación nueva: el asistente aparece después de aceptar el aviso de responsabilidad.
document.addEventListener("ot-terms", () => { if (!(window.BOOT && window.BOOT.wizardDone)) setTimeout(() => showWizard(() => document.dispatchEvent(new Event("ot-wizard"))), 200); });

/* ---------- después de formatear ---------- */
// gpuVendor es el fabricante de la placa de video que maneja el monitor (o la dedicada, si hay integrada).
function gpuVendor() {
  const h = S.hardware || {}, gs = h.gpus || [];
  const g = gs.find(x => x.active) || gs.find(x => x.vendor && x.vendor !== h.cpuVendor) || gs[0];
  return (g && g.vendor) || "amd";
}

const FRESH = [
  ["winupdate", "Actualizá Windows", "Instalá todas las actualizaciones antes que los drivers: así Windows no los reemplaza después por versiones genéricas.", "Abrir Windows Update", () => api("opensettings", { page: "winupdate" })],
  ["chipset", "Driver del chipset", "Bajalo de la web de AMD o Intel según tu procesador. Mejora la administración de energía y los núcleos.", "Abrir página de drivers", () => api("apps/drivers", { vendor: (S.hardware && S.hardware.cpuVendor) || "amd" })],
  ["gpu", "Driver de la placa de video", "Instalá el último driver oficial de tu placa de video, no el que pone Windows Update.", "Abrir página de drivers", () => api("apps/drivers", { vendor: gpuVendor() })],
  ["runtimes", "Runtimes de juegos", "Visual C++, DirectX y .NET: muchos juegos no abren sin ellos.", "Ir a Instalar programas", () => { view = "install"; render(); }],
  ["launchers", "Launchers de juegos", "Steam, Epic, EA, Ubisoft, Battle.net… y tus programas de siempre.", "Ir a Instalar programas", () => { view = "install"; render(); }],
  ["bios", "Revisá la BIOS", "Activá XMP/EXPO para que la RAM vaya a su velocidad, y Resizable BAR si tu equipo lo permite.", "Ir a Diagnóstico", () => { view = "fluidez"; render(); }],
  ["profile", "Aplicá los ajustes recomendados", "El asistente te arma una recomendación según el tipo de juegos que jugás.", "Abrir el asistente", () => showWizard(() => render())],
  ["restore", "Creá un punto de restauración", "Con todo instalado y andando, guardá un punto para volver si algo sale mal.", "Crear punto de restauración", async () => { modal.open("Punto de restauración"); try { await api("restorepoint", {}); modal.log("Punto de restauración creado.", "ok"); } catch (e) { modal.log(e.message, "bad"); } modal.progress(1); modal.done(); }],
];

function renderFresh(m) {
  const marks = (S.settings && S.settings.fresh) || {};
  const done = FRESH.filter(f => marks[f[0]]).length;
  m.innerHTML = `<div class="top"><div><h2>Después de formatear</h2><p>Los pasos para dejar una PC recién instalada lista para jugar, en orden. Marcá cada uno cuando lo termines.</p></div></div>${banners()}
    <div class="banner">${done} de ${FRESH.length} pasos hechos</div>
    <div class="list">${FRESH.map(([id, name, desc, label], i) => `<div class="row ${marks[id] ? "on" : ""}"><div><h4>${i + 1}. ${name}</h4><p>${desc}</p></div><span class="bar"><button class="btn" data-fa="${i}">${label}</button><button class="sw" data-fk="${id}" role="switch" aria-checked="${marks[id] ? "true" : "false"}" aria-label="${name}"></button></span></div>`).join("")}</div>`;
  m.querySelectorAll("[data-fa]").forEach(b => b.onclick = async () => { try { await FRESH[+b.dataset.fa][4](); } catch (e) { toast(e.message, 5000); } });
  m.querySelectorAll("[data-fk]").forEach(b => b.onclick = async () => {
    const on = b.getAttribute("aria-checked") !== "true";
    try { const st = await api("settings", { fresh: { [b.dataset.fk]: on } }); S.settings = st; renderFresh(m); } catch (e) { toast(e.message, 5000); }
  });
}

/* ---------- reportar un problema ---------- */
async function reportIssue() {
  let d;
  try { d = await api("issue?lang=" + (window.UILANG || "es")); } catch (e) { return toast(e.message, 5000); }
  const dlg = v31Dialog(`<h3>Reportar un problema</h3>
    <p class="note">Revisá el texto antes de enviarlo: lleva tu hardware y los avisos del Diagnóstico, sin el nombre del equipo ni tus carpetas. Nada se envía solo: se abre GitHub en tu navegador con esto cargado y vos decidís si lo publicás (necesitás una cuenta de GitHub).</p>
    <label class="fl">Título<input type="text" id="isT" maxlength="120" placeholder="Por ejemplo: el overlay no aparece en CS2" style="width:100%"></label>
    <label class="fl">Descripción<textarea id="isB" rows="14" style="width:100%">${esc(d.body)}</textarea></label>
    <div class="end"><button class="btn" data-c>Cancelar</button><button class="btn" data-copy>Copiar texto</button><button class="btn primary" data-open>Abrir en GitHub</button></div>`);
  const el = dlg.el;
  el.querySelector("[data-c]").onclick = dlg.close;
  el.querySelector("[data-copy]").onclick = async () => { try { await navigator.clipboard.writeText($("#isT").value + "\n\n" + $("#isB").value); toast("Texto copiado."); } catch (e) { toast("No se pudo copiar.", 4000); } };
  el.querySelector("[data-open]").onclick = async () => {
    try { await api("issue", { title: $("#isT").value, body: $("#isB").value }); dlg.close(); toast("Se abrió GitHub en tu navegador."); } catch (e) { toast(e.message, 5000); }
  };
  $("#isT").focus();
}

/* ---------- overlay como imagen ---------- */
// ovPng dibuja la vista previa del overlay en un lienzo (fondo, barras y textos con sus colores) y la
// descarga como PNG.
function ovPng() {
  const root = document.querySelector("#ovPrev > .ovp");
  if (!root) return toast("No hay vista previa para guardar.");
  const box = root.getBoundingClientRect(), k = 2, pad = 16;
  const cv = document.createElement("canvas");
  cv.width = Math.ceil((box.width + pad * 2) * k); cv.height = Math.ceil((box.height + pad * 2) * k);
  const g = cv.getContext("2d");
  g.scale(k, k); g.translate(pad - box.left, pad - box.top);
  const rr = (x, y, w, h, r) => { g.beginPath(); g.moveTo(x + r, y); g.arcTo(x + w, y, x + w, y + h, r); g.arcTo(x + w, y + h, x, y + h, r); g.arcTo(x, y + h, x, y, r); g.arcTo(x, y, x + w, y, r); g.closePath(); };
  const paint = el => {
    const cs = getComputedStyle(el), b = el.getBoundingClientRect();
    if (cs.display === "none" || cs.visibility === "hidden") return;
    const bg = cs.backgroundColor;
    if (bg && bg !== "transparent" && !/rgba\(.*,\s*0\)$/.test(bg)) { g.fillStyle = bg; rr(b.left, b.top, b.width, b.height, Math.min(parseFloat(cs.borderTopLeftRadius) || 0, b.height / 2)); g.fill(); }
    for (const n of el.childNodes) {
      if (n.nodeType === 1) { paint(n); continue; }
      if (n.nodeType !== 3 || !n.data.trim()) continue;
      const r = document.createRange(); r.selectNodeContents(n);
      const t = r.getBoundingClientRect();
      g.font = `${cs.fontStyle} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
      g.fillStyle = cs.color; g.textBaseline = "middle";
      g.fillText(n.data.trim(), t.left, t.top + t.height / 2);
    }
  };
  paint(root);
  cv.toBlob(blob => {
    if (!blob) return toast("No se pudo crear la imagen.", 4000);
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob); a.download = "overdrivetuner-overlay.png";
    document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 4000);
    toast("Imagen guardada en Descargas.");
  }, "image/png");
}

/* ---------- throttling (estado del overlay) ---------- */
function throttleHtml(st) {
  if (!st || !st.throttle) return "";
  const temp = st.throttleTemp > 0 ? ` (${Math.round(st.throttleTemp)} °C)` : "";
  return st.throttle === "cpu"
    ? `<div class="banner warn">El procesador está limitado al ${Math.round(st.throttleLimit)}% de su velocidad máxima${temp}. Se está calentando o el plan de energía lo frena: revisá el disipador y el plan de energía.</div>`
    : `<div class="banner warn">La placa de video lleva un rato por encima de 90 °C${temp} a pleno: puede estar bajando su velocidad. Revisá la ventilación y limpiá los ventiladores.</div>`;
}

/* ---------- latencia del sistema (Diagnóstico) ---------- */
function dpcHtml(r) {
  const tag = { ok: "bien", warn: "aviso", bad: "problema" }[r.status] || r.status;
  return `<div class="list"><div class="row ${r.status === "ok" ? "on" : ""}"><div><h4>Latencia del sistema (DPC) <span class="tag">${tag}</span></h4>
    <p>DPC: ${r.dpcAvg}% promedio, pico ${r.dpcMax}%${r.core ? ` (núcleo ${esc(r.core)})` : ""} · Interrupciones: ${r.intAvg}% promedio, pico ${r.intMax}%</p><p>${esc(r.detail)}</p>
    ${(r.tips || []).length ? `<ul>${r.tips.map(t => `<li>${esc(t)}</li>`).join("")}</ul>` : ""}</div></div></div>`;
}

/* ---------- Limpieza: programas y juegos sin usar, drivers viejos ---------- */
function v31Clean(el) {
  if (!el) return;
  el.innerHTML = `<h3 style="margin-top:22px">Programas y juegos sin usar</h3>
    <p class="note">Lista lo que no abrís hace tiempo y cuánto ocupa. No se desinstala nada solo: el botón abre el desinstalador oficial de cada programa (o Steam).</p>
    <div class="bar"><button class="btn" id="oaGo">Buscar (unos segundos)</button><label class="bar" id="oaAllL" hidden><input type="checkbox" id="oaAll"> Mostrar todos</label></div><div id="oaOut"></div>
    <h3 style="margin-top:22px">Drivers viejos</h3>
    <p class="note">Windows guarda las versiones anteriores de cada driver aunque ya instalaste una más nueva. Acá aparecen solo las que ningún dispositivo usa, ni siquiera uno desconectado.</p>
    <div class="bar"><button class="btn" id="drGo">Buscar drivers viejos</button></div><div id="drOut"></div>`;
  let items = [];
  const ago = it => it.days < 0 ? "sin registro de uso" : it.days === 0 ? "usado hoy" : `hace ${it.days} días`;
  const size = mb => mb >= 1024 ? (mb / 1024).toFixed(1) + " GB" : mb + " MB";
  const paintApps = () => {
    const all = $("#oaAll").checked;
    const l = items.filter(it => all || it.days >= 90 || it.days < 0).slice(0, 80);
    $("#oaOut").innerHTML = l.length ? `<div class="list">${l.map((it, i) => `<div class="row"><div><h4>${esc(it.name)} <span class="tag">${it.kind === "game" ? "juego" : "programa"}</span></h4><p>${it.sizeMB ? size(it.sizeMB) + " · " : ""}${ago(it)}${it.publisher ? " · " + esc(it.publisher) : ""}</p></div><button class="btn" data-un="${items.indexOf(it)}">Desinstalar</button></div>`).join("")}</div>`
      : `<p class="note">No hay programas ni juegos sin usar hace más de 90 días.</p>`;
    $("#oaOut").querySelectorAll("[data-un]").forEach(b => b.onclick = async () => {
      const it = items[+b.dataset.un];
      if (!await ask(`Se va a abrir el desinstalador de «${it.name}». Seguí los pasos ahí; si cambiás de idea, cancelalo en esa ventana.`, { title: "Desinstalar", ok: "Abrir desinstalador" })) return;
      try { await api("oldapps/uninstall", { id: it.id }); toast("Se abrió el desinstalador."); } catch (e) { toast(e.message, 6000); }
    });
  };
  $("#oaGo").onclick = async () => {
    const b = $("#oaGo"); b.disabled = true; $("#oaOut").innerHTML = '<p class="note">Buscando…</p>';
    try { items = await api("oldapps"); $("#oaAllL").hidden = false; paintApps(); } catch (e) { $("#oaOut").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; }
    b.disabled = false;
  };
  $("#oaAll").onchange = paintApps;
  $("#drGo").onclick = async () => {
    const b = $("#drGo"); b.disabled = true; $("#drOut").innerHTML = '<p class="note">Buscando…</p>';
    let d;
    try { d = await api("drivers/old"); } catch (e) { $("#drOut").innerHTML = `<div class="banner bad">${esc(e.message)}</div>`; b.disabled = false; return; }
    b.disabled = false;
    if (d.unsupported) { $("#drOut").innerHTML = '<p class="note">Tu versión de Windows no permite listar los drivers de forma segura (hace falta Windows 11 24H2 o más nuevo).</p>'; return; }
    const l = d.drivers || [];
    if (!l.length) { $("#drOut").innerHTML = '<p class="note">No hay drivers viejos para borrar.</p>'; return; }
    const tot = l.reduce((a, p) => a + p.sizeMB, 0);
    $("#drOut").innerHTML = `<table class="table"><tbody>${l.map((p, i) => `<tr><td><input type="checkbox" class="drk" data-i="${i}" checked aria-label="${esc(p.original)}"></td><td><b>${esc(p.provider)}</b> · ${esc(p.class)}<div class="note">${esc(p.original)} ${esc(p.version)} (${esc(p.date)}) → reemplazado por ${esc(p.newer)}</div></td><td>${p.sizeMB ? size(p.sizeMB) : "—"}</td></tr>`).join("")}</tbody></table>
      <div class="bar" style="margin-top:10px"><button class="btn danger" id="drDel" ${S.admin ? "" : "disabled"}>Borrar los seleccionados</button><span class="sp">${l.length} paquete(s) · ${size(tot)}</span></div>`;
    $("#drDel").onclick = async () => {
      const sel = [...$("#drOut").querySelectorAll(".drk:checked")].map(c => l[+c.dataset.i]);
      if (!sel.length) return toast("Seleccioná qué borrar.");
      if (!await ask("Se van a borrar del almacén de Windows las versiones viejas elegidas. Ningún dispositivo las usa, y si alguno las usara Windows se niega a borrarlas.\n\nNo se pueden recuperar desde acá: para volver a una versión vieja habría que bajarla de nuevo.", { title: "Borrar drivers viejos", ok: "Borrar", danger: true })) return;
      modal.open("Borrando drivers viejos");
      try {
        const res = await api("drivers/delete", { names: sel.map(p => p.name), confirm: 1 });
        res.forEach(r => { const p = sel.find(x => x.name === r.name); modal.log((r.ok ? "✔ " : "✖ ") + p.original + " " + p.version + (r.ok ? "" : ": " + r.error), r.ok ? "ok" : "bad"); });
      } catch (e) { modal.log(e.message, "bad"); }
      modal.progress(1); modal.done();
    };
  };
}

/* ---------- Modo juego: notificaciones ---------- */
function gmQuiet(el, cfg, save) {
  if (!el) return;
  el.innerHTML = `<h3 style="margin-top:22px">Notificaciones</h3><div class="list"><div class="row"><div><h4>Silenciar las notificaciones mientras jugás</h4><p>Al abrir un juego se apagan los avisos emergentes de Windows y al cerrarlo vuelven como estaban. Si igual te aparecen, activá también la opción propia de Windows («No molestar» automático al jugar).</p></div><span class="bar"><button class="btn" id="gmQuietWin">Abrir la opción de Windows</button><button class="sw" id="gmQuietOn" role="switch" aria-checked="${cfg.quietGame ? "true" : "false"}" aria-label="Silenciar las notificaciones mientras jugás"></button></span></div></div>`;
  $("#gmQuietOn").onclick = () => { cfg.quietGame = !cfg.quietGame; $("#gmQuietOn").setAttribute("aria-checked", cfg.quietGame); save(); };
  $("#gmQuietWin").onclick = () => api("opensettings", { page: "notifications" }).catch(e => toast(e.message, 5000));
}

/* ---------- Configuración: asistente, recordatorios y reporte ---------- */
function v31Settings(m) {
  const st = S.settings || {};
  m.insertAdjacentHTML("beforeend", `<h3 style="margin-top:22px">Ayuda y mantenimiento</h3><div class="list">
    <div class="row"><div><h4>Recordatorios de mantenimiento</h4><p>Un aviso en la bandeja si hace más de un mes que no limpiás los temporales o si tu driver de video tiene más de 9 meses. Como mucho uno cada dos semanas.</p></div><button class="sw" id="rmOn" role="switch" aria-checked="${st.remindOff ? "false" : "true"}" aria-label="Recordatorios de mantenimiento"></button></div>
    <div class="row"><div><h4>Asistente de ajustes</h4><p>Te pregunta qué tipo de juegos jugás y arma una recomendación de ajustes seguros.</p></div><button class="btn" id="wzAgain">Abrir el asistente</button></div>
    <div class="row"><div><h4>Reportar un problema</h4><p>Arma un reporte con tu hardware y el Diagnóstico para abrir un issue en GitHub. Lo revisás antes de enviarlo.</p></div><button class="btn" id="isGo">Reportar un problema</button></div>
  </div>`);
  $("#rmOn").onclick = async () => {
    const on = $("#rmOn").getAttribute("aria-checked") !== "true";
    try { S.settings = await api("settings", { remindOff: !on }); $("#rmOn").setAttribute("aria-checked", on); } catch (e) { toast(e.message, 5000); }
  };
  $("#wzAgain").onclick = () => showWizard();
  $("#isGo").onclick = reportIssue;
}
