"use strict";
/* Modo juego: programas que se cierran siempre al abrir un juego. */

/* Programas que no están protegidos pero que suelen ser parte del juego o de su anti-cheat: se pide confirmación. */
const GM_SENSITIVE = ["steam.exe", "steamwebhelper.exe", "epicgameslauncher.exe", "eadesktop.exe", "origin.exe", "battle.net.exe", "galaxyclient.exe", "ubisoftconnect.exe", "upc.exe", "faceit.exe", "faceitclient.exe", "faceitac.exe", "easyanticheat.exe", "vgc.exe", "beservice.exe", "riotclientservices.exe"];
const GM_PROTECTED = ["system", "idle", "registry", "smss", "csrss", "wininit", "winlogon", "services", "lsass", "svchost", "dwm", "explorer", "fontdrvhost", "sihost", "taskhostw", "ctfmon", "audiodg", "searchindexer", "runtimebroker", "msmpeng", "dllhost", "wudfhost", "spoolsv", "securityhealthservice", "startmenuexperiencehost", "shellexperiencehost", "textinputhost", "conhost", "applicationframehost", "overdrivetuner", "fpsforge"];

function gmName(v) {
  v = String(v || "").trim();
  if (!v) return "";
  return /\.exe$/i.test(v) ? v : v + ".exe";
}

async function renderGameMode(m) {
  const head = `<div class="top"><div><h2>Modo juego</h2><p>Elegí los programas que querés que se cierren sí o sí cuando se abre un juego, para liberar procesador y memoria. Se cierran todos sus procesos apenas el overlay detecta el juego, y no se vuelven a abrir solos. Nunca se toca el juego ni los procesos de Windows.</p></div></div>`;
  let d;
  try { d = await api("overlay"); } catch (e) { m.innerHTML = head + `<div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "gamemode") return;
  const cfg = d.config;
  cfg.closeList = cfg.closeList || [];
  const save = async () => { try { Object.assign(cfg, await api("overlay/config", cfg)); } catch (e) { toast(e.message, 6000); const n = await api("overlay"); Object.assign(cfg, n.config); } paint(); };
  m.innerHTML = head + banners() +
    (d.running ? "" : `<div class="banner">El Modo juego funciona junto con el overlay, que es el que detecta el juego, y ahora no está abierto. <button class="btn" id="gmStart">Abrir overlay</button></div>`) +
    `<h3>Se cierran al abrir un juego</h3><div class="list" id="gmList"></div>
    <div class="gm-add"><input type="text" id="gmNew" placeholder="programa, ej. chrome.exe" style="width:240px" aria-label="Programa a cerrar"><button class="btn" id="gmAdd">Agregar</button><button class="btn" id="gmPick">Elegir de los programas abiertos</button></div>
    <div id="gmPickOut"></div>`;
  const paint = () => {
    const l = cfg.closeList;
    $("#gmList").innerHTML = l.length
      ? l.map((n, i) => `<div class="row"><div><h4>${esc(n)}</h4></div><button class="btn" data-del="${i}">Quitar</button></div>`).join("")
      : '<div class="row"><div><p>Todavía no agregaste ningún programa. Por ejemplo: un navegador, Spotify o un cliente de descargas.</p></div></div>';
    $("#gmList").querySelectorAll("[data-del]").forEach(b => b.onclick = () => { cfg.closeList.splice(+b.dataset.del, 1); save(); });
  };
  const add = async raw => {
    const n = gmName(raw);
    if (!n) return;
    if (!/^[^\\/:*?"<>| ,;&]{1,60}\.exe$/i.test(n)) return toast("Poné solo el nombre del programa, por ejemplo chrome.exe", 5000);
    const base = n.replace(/\.exe$/i, "").toLowerCase();
    if (GM_PROTECTED.includes(base)) return toast("«" + n + "» es parte de Windows o de OverdriveTuner y no se puede cerrar desde acá.", 6000);
    if (cfg.closeList.some(x => x.toLowerCase() === n.toLowerCase())) return toast("Ya está en la lista.");
    if (GM_SENSITIVE.includes(n.toLowerCase()) && !await ask(`«${n}» suele ser un launcher o parte del anti-cheat de un juego. Si lo cerrás al abrir el juego, el juego puede cerrarse o no dejarte entrar a la partida.\n\n¿Agregarlo igual?`, { title: "Programa delicado", ok: "Agregar igual", danger: true })) return;
    cfg.closeList.push(n);
    await save();
  };
  $("#gmAdd").onclick = async () => { await add($("#gmNew").value); $("#gmNew").value = ""; };
  $("#gmNew").onkeydown = e => { if (e.key === "Enter") $("#gmAdd").click(); };
  if ($("#gmStart")) $("#gmStart").onclick = async () => { try { await api("overlay/start", {}); toast("Abriendo overlay…"); setTimeout(() => renderGameMode(m), 1800); } catch (e) { toast(e.message, 6000); } };
  $("#gmPick").onclick = async e => {
    const b = e.currentTarget; b.disabled = true;
    $("#gmPickOut").innerHTML = '<p class="note">Midiendo qué programas están abiertos (2 segundos)…</p>';
    try {
      const ps = (await api("procs")).filter(p => p.safe && !cfg.closeList.some(x => x.toLowerCase() === (p.name + ".exe").toLowerCase()));
      $("#gmPickOut").innerHTML = ps.length
        ? `<div class="list">${ps.map(p => `<div class="row"><div><h4>${esc(p.name)}.exe</h4><p>${p.mb} MB de memoria</p></div><button class="btn" data-pick="${esc(p.name)}">Agregar</button></div>`).join("")}</div>`
        : '<p class="note">No hay programas para sugerir ahora.</p>';
      $("#gmPickOut").querySelectorAll("[data-pick]").forEach(x => x.onclick = async () => { await add(x.dataset.pick); x.closest(".row").remove(); });
    } catch (err) { $("#gmPickOut").innerHTML = `<div class="banner bad">${esc(err.message)}</div>`; }
    b.disabled = false;
  };
  paint();
}
