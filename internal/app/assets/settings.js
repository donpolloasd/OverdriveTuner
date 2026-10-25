"use strict";
/* Pestaña Configuración: idioma, atajos globales, importar/exportar, animación de apertura. */

const HK_LABELS = [["hkShow", "Mostrar / ocultar el overlay", "Ctrl+Alt+O"], ["hkRecord", "Empezar / detener grabación", "Ctrl+Alt+R"], ["hkCorner", "Cambiar de esquina", "Ctrl+Alt+P"]];
const HK_KEYS_UNUSED = { " ": "Space", PageUp: "PageUp", PageDown: "PageDown", End: "End", Home: "Home", ArrowLeft: "Left", ArrowRight: "Right", ArrowUp: "Up", ArrowDown: "Down", Insert: "Insert", Delete: "Delete" };

/* Convierte un evento de teclado en texto de atajo, o "" si todavía falta la tecla. */
function hkFromEvent(e) {
  let k = e.key;
  if (["Control", "Alt", "Shift", "Meta", "AltGraph"].includes(k)) return "";
  let name = HK_KEYS[k] || "";
  if (!name && /^F([1-9]|1[0-2])$/.test(k)) name = k;
  if (!name && /^[a-z0-9]$/i.test(k)) name = k.toUpperCase();
  if (!name && /^(Key[A-Z]|Digit[0-9])$/.test(e.code)) name = e.code.slice(-1);
  if (!name) return "";
  const mods = [];
  if (e.ctrlKey) mods.push("Ctrl");
  if (e.altKey) mods.push("Alt");
  if (e.shiftKey) mods.push("Shift");
  if (e.metaKey) mods.push("Win");
  if (!mods.length) return "";
  return mods.concat(name).join("+");
}

const HK_KEYLIST = [..."ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"].concat(Array.from({ length: 12 }, (_, i) => "F" + (i + 1)), ["Space", "Home", "End", "PageUp", "PageDown", "Insert", "Delete", "Left", "Right", "Up", "Down"]);
function hkParts(v) { const p = String(v || "").split("+"); const key = p.pop() || "O"; return { mods: p, key }; }
function hkRow(k, n, v) {
  const { mods, key } = hkParts(v);
  return `<div class="row" data-hkrow="${k}"><div><h4>${n}</h4><p>Elegí los modificadores y la tecla. Se necesitan dos modificadores, o uno y una tecla F1–F12.</p></div><div class="bar">${["Ctrl", "Alt", "Shift", "Win"].map(x => `<label><input type="checkbox" data-mod="${x}" ${mods.includes(x) ? "checked" : ""}> ${x}</label>`).join(" ")}
    <select data-key aria-label="Tecla">${HK_KEYLIST.map(x => `<option value="${x}" ${x === key ? "selected" : ""}>${x}</option>`).join("")}</select><button class="btn" data-hkreset="${k}">Restablecer</button></div></div>`;
}
function hkSet(m, k, v) {
  const row = m.querySelector(`[data-hkrow="${k}"]`); if (!row) return;
  const { mods, key } = hkParts(v);
  row.querySelectorAll("[data-mod]").forEach(c => c.checked = mods.includes(c.dataset.mod));
  row.querySelector("[data-key]").value = key;
}

function introPrefs() {
  const s = (S && S.settings) || {};
  return { off: !!s.introOff, vol: Number.isFinite(+s.introVol) ? +s.introVol : 70 };
}
function cacheIntroPrefs(p) { lsSet("otIntroOff", p.off ? "1" : "0"); lsSet("otIntroVol", String(p.vol)); }

async function saveSettings(patch) {
  const r = await api("settings", patch);
  if (S) S.settings = Object.assign({}, S.settings, r);
  return r;
}

async function renderSettings(m) {
  let d;
  try { d = await api("overlay"); } catch (e) { m.innerHTML = `<div class="top"><div><h2>Configuración</h2></div></div><div class="banner bad">${esc(e.message)}</div>`; return; }
  if (view !== "settings") return;
  const cfg = d.config, st = S.settings || {}, ip = introPrefs();
  const lang = UILANG;
  let au = { enabled: false };
  try { au = await api("autostart"); } catch (e) { /* sin dato */ }
  const errs = (d.state && d.state.hotkeyErrors) || [];
  m.innerHTML = `<div class="top"><div><h2>Configuración</h2><p>Idioma, atajos de teclado, copia de tu configuración y animación de apertura.</p></div></div>${banners()}
  <h3 id="stLangH">Idioma</h3>
  <div class="row" id="stLang"><div><h4>Idioma de la interfaz</h4><p>Cambia todos los textos de la aplicación. El panel que se dibuja encima del juego (overlay) mantiene sus textos en español.</p></div>
   <select id="sLang"><option value="es" ${lang === "es" ? "selected" : ""}>Español</option><option value="en" ${lang === "en" ? "selected" : ""}>English</option></select></div>

  <h3 id="stHkH" style="margin-top:22px">Atajos globales</h3>
  <div class="list" id="stHk">
   <div class="row"><div><h4>Atajos activados</h4><p>Funcionan aunque estés dentro del juego. Si otro programa ya usa una combinación, no se registra y te avisamos acá.</p></div><button class="sw" id="hkOn" role="switch" aria-checked="${cfg.hotkey ? "true" : "false"}" aria-label="Atajos activados"></button></div>
   ${HK_LABELS.map(([k, n]) => hkRow(k, n, cfg[k])).join("")}
   <div id="hkMsg">${errs.map(e => `<div class="banner bad">${esc(e)}</div>`).join("")}</div>
  </div>

  <h3 id="stCfgH" style="margin-top:22px">Copia de la configuración</h3>
  <div class="list" id="stCfg">
   <div class="row"><div><h4>Exportar</h4><p>Guarda en un archivo tus ajustes, la configuración del overlay y qué optimizaciones tenés aplicadas. Sirve para tener una copia o llevarla a otra PC.</p></div><button class="btn primary" id="cfgExp">Exportar configuración</button></div>
   <div class="row"><div><h4>Importar</h4><p>Elegí un archivo exportado. Primero te mostramos qué se va a hacer. Los ajustes de anti-cheat y los avanzados nunca se aplican solos desde un archivo.</p></div><span><button class="btn" id="cfgImp">Importar configuración</button><input type="file" id="cfgFile" accept=".json,application/json" hidden></span></div>
   <div id="cfgOut"></div>
  </div>

  <h3 id="stIntroH" style="margin-top:22px">Animación de apertura</h3>
  <div class="list" id="stIntro">
   <div class="row"><div><h4>Mostrar animación al abrir</h4><p>Una intro corta con sonido. Se puede saltar con cualquier tecla o clic.</p></div><button class="sw" id="inOn" role="switch" aria-checked="${ip.off ? "false" : "true"}" aria-label="Mostrar animación al abrir"></button></div>
   <div class="row"><div><h4>Volumen</h4><p>0 la deja muda.</p></div><div class="bar"><input type="range" id="inVol" min="0" max="100" step="5" value="${ip.vol}" aria-label="Volumen"><b id="inVolN" style="min-width:42px;text-align:right">${ip.vol}%</b></div></div>
   <div class="row"><div><h4>Probar</h4><p>La reproduce ahora con el volumen elegido.</p></div><button class="btn" id="inTest">Reproducir animación</button></div>
  </div>

  <h3 id="stBootH" style="margin-top:22px">Inicio con Windows</h3>
  <div class="row" id="stBoot"><div><h4>Iniciar el overlay con Windows</h4><p>Al encender la PC el overlay arranca solo, en segundo plano, y aparece cuando abrís un juego. Mientras no hay juego no se ve ninguna ventana: solo un ícono en la bandeja junto al reloj. Usa una tarea programada con permisos de administrador, así que Windows no pide confirmación cada vez. Si lo apagás, el overlay solo se abre cuando abrís OverdriveTuner.</p><p class="note" id="bootSt"></p></div><span class="bar"><button class="btn" id="bootRun">Probar ahora</button><button class="sw" id="bootOn" role="switch" aria-checked="${au.enabled ? "true" : "false"}" aria-label="Iniciar el overlay con Windows"></button></span></div>

  <h3 id="stWinH" style="margin-top:22px">Ventana</h3>
  <div class="row" id="stWin"><div><h4>Al tocar la X</h4><p>La ventana se cierra pero OverdriveTuner sigue activo en la bandeja, junto al reloj (el overlay y las grabaciones no se cortan). Hacé clic en su ícono para volver a abrirlo. Para cerrarlo del todo usá este botón o el clic derecho del ícono.</p></div><button class="btn danger" id="quitAll">Salir por completo</button></div>

  <h3 id="stTourH" style="margin-top:22px">Ayuda</h3>
  <div class="row" id="stTour"><div><h4>Recorrido guiado</h4><p>Volvé a ver para qué sirve cada sección y cada botón.</p></div><button class="btn" id="tourAgain">Repetir recorrido</button></div>`;

  // idioma
  $("#sLang").onchange = async e => {
    const v = e.target.value;
    try { await saveSettings({ lang: v }); applyLang(v); S = await api("state"); render(); toast(v === "en" ? "Language changed." : "Idioma cambiado."); }
    catch (err) { toast(err.message, 5000); }
  };

  // inicio con Windows
  const bootText = a => {
    if (!a.enabled) return "La tarea de inicio no está registrada.";
    let t = "Tarea registrada";
    if (a.lastRun) t += ". Última vez: " + a.lastRun + (a.lastResult === 0 ? " (se ejecutó bien)" : " (terminó con el código " + a.lastResult + ")");
    else t += ". Todavía no se ejecutó (corre al próximo inicio de sesión)";
    return t + ". Overlay activo ahora: " + (a.overlayRunning ? "sí." : "no.");
  };
  $("#bootSt").textContent = bootText(au);
  $("#bootOn").onclick = async e => {
    const b = e.currentTarget, want = b.getAttribute("aria-checked") !== "true";
    b.disabled = true;
    try { const r = await api("autostart", { enabled: want }); b.setAttribute("aria-checked", r.enabled ? "true" : "false"); $("#bootSt").textContent = bootText(r); toast(r.enabled ? "El overlay arrancará con Windows." : "El overlay ya no arranca con Windows."); }
    catch (err) { toast(err.message, 6000); }
    b.disabled = false;
  };
  $("#bootRun").onclick = async e => {
    const b = e.currentTarget; b.disabled = true;
    try { await api("autostart", { run: true }); await new Promise(r => setTimeout(r, 2500)); const r = await api("autostart"); $("#bootSt").textContent = bootText(r); toast(r.overlayRunning ? "El overlay está activo." : "Se ejecutó la tarea, pero el overlay no aparece activo."); }
    catch (err) { toast(err.message, 6000); }
    b.disabled = false;
  };

  // atajos
  const saveHk = async () => {
    try { Object.assign(cfg, await api("overlay/config", cfg)); }
    catch (err) { toast(err.message, 6000); const n = await api("overlay"); Object.assign(cfg, n.config); }
    HK_LABELS.forEach(([k]) => hkSet(m, k, cfg[k]));
    setTimeout(async () => { if (view !== "settings") return; try { const n = await api("overlay"); $("#hkMsg").innerHTML = ((n.state && n.state.hotkeyErrors) || []).map(e => `<div class="banner bad">${esc(e)}</div>`).join(""); } catch (e) { /* sigue */ } }, 1500);
  };
  $("#hkOn").onclick = e => { cfg.hotkey = !cfg.hotkey; e.currentTarget.setAttribute("aria-checked", cfg.hotkey ? "true" : "false"); saveHk(); };
  const hkRead = k => {
    const row = m.querySelector(`[data-hkrow="${k}"]`);
    const mods = [...row.querySelectorAll("[data-mod]")].filter(c => c.checked).map(c => c.dataset.mod);
    return mods.concat(row.querySelector("[data-key]").value).join("+");
  };
  m.querySelectorAll("[data-hkrow]").forEach(row => {
    const k = row.dataset.hkrow;
    row.querySelectorAll("[data-mod],[data-key]").forEach(el => el.onchange = () => { cfg[k] = hkRead(k); saveHk(); });
  });
  m.querySelectorAll("[data-hkreset]").forEach(b => b.onclick = () => { const row = HK_LABELS.find(x => x[0] === b.dataset.hkreset); cfg[row[0]] = row[2]; saveHk(); });

  // exportar / importar
  $("#cfgExp").onclick = async () => {
    try {
      const r = await fetch("/api/config/export", { headers: { "X-Token": window.TOKEN } });
      if (!r.ok) throw new Error("Error " + r.status);
      const blob = await r.blob(), a = document.createElement("a");
      a.href = URL.createObjectURL(blob); a.download = "overdrivetuner-config.json";
      document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 4000);
      toast("Configuración exportada.");
    } catch (err) { toast(err.message, 5000); }
  };
  $("#cfgImp").onclick = () => $("#cfgFile").click();
  $("#cfgFile").onchange = async e => {
    const f = e.target.files[0]; e.target.value = "";
    if (!f) return;
    let bundle;
    try { bundle = JSON.parse(await f.text()); } catch (err) { toast("El archivo no es un JSON válido.", 5000); return; }
    let plan;
    try { plan = await api("config/import", { bundle, apply: false }); } catch (err) { $("#cfgOut").innerHTML = `<div class="banner bad">${esc(err.message)}</div>`; return; }
    const li = (a, f2) => a.length ? `<ul>${a.map(x => `<li>${esc(x.name)}${f2 && x.reason ? ` <span class="note">— ${esc(x.reason)}</span>` : ""}</li>`).join("")}</ul>` : "";
    $("#cfgOut").innerHTML = `<div class="row" style="display:block"><h4>Esto es lo que se va a hacer</h4>
      <p>Se guardarán los ajustes${plan.overlay ? " y la configuración del overlay" : ""} del archivo.</p>
      <p><b>Optimizaciones a aplicar (${plan.toApply.length})</b></p>${li(plan.toApply) || '<p class="note">Ninguna.</p>'}
      ${plan.already.length ? `<p><b>Ya aplicadas (${plan.already.length})</b></p>${li(plan.already)}` : ""}
      ${plan.skipped.length ? `<p><b>Se omiten (${plan.skipped.length})</b></p>${li(plan.skipped, true)}` : ""}
      <div class="btns" style="margin-top:10px"><button class="btn primary" id="cfgGo">Importar</button><button class="btn" id="cfgNo">Cancelar</button></div></div>`;
    $("#cfgNo").onclick = () => { $("#cfgOut").innerHTML = ""; };
    $("#cfgGo").onclick = async () => {
      try {
        if (plan.toApply.length && !S.admin) throw new Error("Para aplicar optimizaciones el programa necesita permisos de administrador.");
        const done = await api("config/import", { bundle, apply: true });
        S = await api("state");
        const nl = effectiveLang(S.settings && S.settings.lang);
        if (nl !== UILANG) { applyLang(nl); S = await api("state"); }
        cacheIntroPrefs(introPrefs());
        if (plan.toApply.length) { render(); await runIds("Importar configuración", plan.toApply.map(x => x.id), "apply", true); render(); }
        else { toast("Configuración importada."); render(); }
        void done;
      } catch (err) { toast(err.message, 6000); }
    };
  };

  // animación de apertura
  const saveIntro = async patch => { try { await saveSettings(patch); cacheIntroPrefs(introPrefs()); } catch (err) { toast(err.message, 5000); } };
  $("#inOn").onclick = e => { const on = e.currentTarget.getAttribute("aria-checked") !== "true"; e.currentTarget.setAttribute("aria-checked", on ? "true" : "false"); saveIntro({ introOff: !on }); };
  $("#inVol").oninput = e => { $("#inVolN").textContent = e.target.value + "%"; };
  $("#inVol").onchange = e => saveIntro({ introVol: +e.target.value });
  $("#inTest").onclick = () => { if (window.OTIntro) window.OTIntro.play({ vol: +$("#inVol").value, test: true }); };
  $("#quitAll").onclick = async () => { if (!await ask("¿Cerrar OverdriveTuner por completo? También se cierra el overlay.", { title: "Cerrar programa", ok: "Cerrar", danger: true })) return; try { await api("quit", {}); } catch (e) { /* se cierra */ } };
  $("#tourAgain").onclick = () => { view = "home"; render(); setTimeout(() => startTour("home"), 150); };
}
