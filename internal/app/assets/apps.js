"use strict";
/* Pestaña «Instalar programas»: instala y actualiza programas con winget. */

let APPS = null, APPS_LANG = "", APPS_HAVE = null;

async function renderApps(m) {
  const head = `<div class="top"><div><h2>Instalar programas</h2><p>Instalá lo que necesitás con un clic usando winget, el administrador de paquetes de Windows 11. Marcá los programas y tocá «Instalar seleccionados».</p></div></div>`;
  m.innerHTML = head + '<p class="note">Cargando…</p>';
  try {
    if (!APPS || APPS_LANG !== window.UILANG || !APPS.hwReady) { APPS = await api("apps?lang=" + (window.UILANG || "es")); APPS_LANG = window.UILANG; APPS_HAVE = null; }
  } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "install") return;
  const wg = APPS.winget;
  const rows = c => APPS.items.filter(x => x.cat === c.id).map(x => x.url
    ? `<tr data-id="${esc(x.id)}"><td style="width:34px"></td><td><b>${esc(x.name)}</b>
        <div class="cmd">${esc(x.desc)}${x.extra ? " " + esc(x.extra) : ""}</div></td><td style="width:150px;text-align:right"><button class="btn" data-open="${esc(x.id)}">Abrir página oficial</button></td></tr>`
    : `<tr data-id="${esc(x.id)}">
      <td style="width:34px"><input type="checkbox" class="ck" data-id="${esc(x.id)}" ${wg && S.admin ? "" : "disabled"} aria-label="${esc(x.name)}"></td>
      <td><b>${esc(x.name)}</b> <span class="tag ok" data-st="${esc(x.id)}" hidden>Instalado</span>
        <div class="cmd">${esc(x.desc)}${x.extra ? " " + esc(x.extra) : ""}</div></td>
      <td style="width:150px;text-align:right"><button class="btn" data-open="${esc(x.id)}" data-site="1" ${wg ? "hidden" : ""}>Abrir web oficial</button></td></tr>`).join("");
  const vend = APPS.gpuVendors.length ? APPS.gpuVendors : ["nvidia", "amd", "intel"];
  const vname = { nvidia: "NVIDIA", amd: "AMD", intel: "Intel" };
  m.innerHTML = head + banners() +
    (wg ? "" : `<div class="banner bad">winget no está disponible en este equipo. Es el administrador de paquetes de Windows 11 que usa esta sección. <button class="btn" id="getWinget">Instalar winget</button></div>`) +
    APPS.categories.map(c => `<h3 style="margin-top:20px">${esc(c.name)}</h3><table class="table"><tbody>${rows(c)}</tbody></table>`).join("") +
    `<div class="bar" style="margin-top:16px;flex-wrap:wrap;gap:8px">
      <button class="btn primary" id="appsInstall" ${wg && S.admin ? "" : "disabled"}>Instalar seleccionados</button>
      <button class="btn" id="appsUpgrade" ${wg && S.admin ? "" : "disabled"}>Actualizar programas instalados</button>
      <span class="sp"></span><span class="note" id="appsNote">Consultando qué ya está instalado…</span></div>
    <h3 style="margin-top:22px">Drivers de video</h3>
    <p class="note">Abre la página oficial del fabricante en tu navegador. Los drivers no se instalan desde acá: descargalos y verificalos vos.</p>
    <div class="bar">${vend.map(v => `<button class="btn drv" data-v="${v}">Drivers de ${vname[v]}</button>`).join("")}</div>`;

  if ($("#getWinget")) $("#getWinget").onclick = async () => {
    const b = $("#getWinget"); b.disabled = true;
    try { await api("apps/winget", {}); APPS = null; toast("winget listo."); renderApps(m); }
    catch (e) { toast("No se pudo: " + e.message, 8000); b.disabled = false; }
  };
  m.querySelectorAll("[data-open]").forEach(b => b.onclick = async () => {
    try { await api("apps/open", { id: b.dataset.open }); } catch (e) { toast("No se pudo abrir: " + e.message, 6000); }
  });
  m.querySelectorAll(".drv").forEach(b => b.onclick = async () => {
    try { await api("apps/drivers", { vendor: b.dataset.v }); } catch (e) { toast("No se pudo abrir: " + e.message, 6000); }
  });
  $("#appsInstall").onclick = () => appsInstallSelected(m);
  $("#appsUpgrade").onclick = () => appsUpgradeAll(m);
  if (wg) appsLoadInstalled(m, false);
  else $("#appsNote").textContent = "";
}

async function appsLoadInstalled(m, fresh) {
  const note = $("#appsNote");
  try {
    APPS_HAVE = await api("apps/installed" + (fresh ? "?fresh=1" : ""));
  } catch (e) { if (note) note.textContent = "No se pudo consultar qué está instalado."; return; }
  if (view !== "install") return;
  let n = 0;
  m.querySelectorAll("[data-st]").forEach(t => { const on = !!APPS_HAVE[t.dataset.st]; t.hidden = !on; if (on) n++; });
  m.querySelectorAll(".ck").forEach(c => { if (APPS_HAVE[c.dataset.id]) { c.checked = false; c.disabled = true; } });
  if (note) note.textContent = `${n} de ${APPS.items.filter(x => !x.url).length} ya instalados`;
}

async function appsInstallSelected(m) {
  if (busyOp) return;
  const ids = [...m.querySelectorAll(".ck:checked")].map(c => c.dataset.id);
  if (!ids.length) return toast("Marcá qué instalar.");
  const items = ids.map(id => APPS.items.find(x => x.id === id));
  busyOp = true; modal.open("Instalando programas");
  let ok = 0, bad = 0;
  for (let i = 0; i < items.length; i++) {
    const it = items[i];
    modal.log("Instalando " + it.name + "…", "note");
    try {
      const r = await api("apps/install", { id: it.id });
      if (r.ok) { ok++; modal.log("  listo", "ok"); }
      else { bad++; (r.notes || []).forEach(x => modal.log("  " + x, "bad")); showSiteBtn(m, it.id); modal.log("  Podés descargarlo de la web oficial con el botón de su fila.", "note"); }
    } catch (e) { bad++; modal.log("  " + e.message, "bad"); showSiteBtn(m, it.id); }
    modal.progress((i + 1) / items.length);
  }
  modal.log(`Terminado: ${ok} instalados, ${bad} con problemas.`, bad ? "bad" : "ok");
  modal.done(); busyOp = false;
  appsLoadInstalled(m, true);
}

function showSiteBtn(m, id) {
  const b = m.querySelector(`[data-open="${id}"][data-site]`);
  if (b) b.hidden = false;
}

async function appsUpgradeAll(m) {
  if (busyOp) return;
  if (!await ask("Se van a actualizar todos los programas instalados que winget pueda actualizar. Puede tardar varios minutos. ¿Seguimos?", { title: "Actualizar programas", ok: "Actualizar todo" })) return;
  busyOp = true; modal.open("Actualizando programas");
  modal.log("winget está actualizando tus programas… no cierres esta ventana.", "note");
  try { await api("apps/upgrade", {}); modal.log("Listo: todo lo que se podía actualizar quedó al día.", "ok"); }
  catch (e) { modal.log(e.message, "bad"); }
  modal.progress(1); modal.done(); busyOp = false;
  appsLoadInstalled(m, true);
}
