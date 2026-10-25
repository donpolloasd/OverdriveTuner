"use strict";
/* Guía animada «Cómo se miden los FPS»: explica que el overlay usa RTSS para los FPS y Core Temp para la temperatura del procesador. */

function openFpsHelp() {
  if (document.getElementById("fhGuide")) return;
  const flow = (a, mid, b, cls = "") => `<div class="fh-flow ${cls}">
      <div class="fh-box">${a}</div><div class="fh-link"><i></i></div>
      ${mid ? `<div class="fh-box fh-mid">${mid}</div><div class="fh-link"><i></i></div>` : ""}
      <div class="fh-box fh-on">${b}</div></div>`;
  const scenes = [
    {
      t: "Quién mide y quién muestra",
      art: flow('<b>El juego</b><span class="fh-in">Acá adentro cuenta RTSS</span>', "<b>Memoria compartida</b><span>FPS y temperaturas</span>", "<b>OverdriveTuner</b><span>solo lee y dibuja</span>"),
      body: `<p>Contar los cuadros de un juego solo se puede hacer desde adentro de ese juego. Eso lo hace RTSS (RivaTuner Statistics Server), que se mete dentro del juego, y deja los números en una memoria compartida de Windows.</p>
             <p>OverdriveTuner en sí es externo: no entra al juego, no lo modifica ni le inyecta nada. Solo lee esos números desde afuera y los dibuja en su propia ventana. Lo que entra al juego es RTSS, no OverdriveTuner.</p>`,
    },
    {
      t: "Por qué hace falta RTSS",
      art: `<div class="fh-two">${flow("<b>OverdriveTuner</b>", "", "<b>El juego</b>", "fh-blocked")}<small>Sin RTSS, desde afuera no hay nada que leer.</small>
            ${flow('<b>El juego</b><span class="fh-in">RTSS</span>', "<b>Memoria compartida</b>", "<b>OverdriveTuner</b>")}<small>Con RTSS, los números llegan solos.</small></div>`,
      body: `<p>Los FPS salen siempre de RTSS. Es liviano, queda en segundo plano y no hace falta MSI Afterburner. El Setup lo instala por vos y el overlay lo abre solo.</p>
             <p>La temperatura del procesador sale de Core Temp, que también instala el Setup y el overlay abre solo. Windows no la entrega sin un driver, y OverdriveTuner no instala drivers para no chocar con los anti-cheat. Core Temp solo la publica y OverdriveTuner la lee.</p>`,
    },
    {
      t: "Cómo dejarlo andando",
      art: "",
      body: `<ol class="fh-steps">
        <li><b>Instalá RTSS y Core Temp.</b> El Setup y el Parche tienen una casilla para cada uno. También están en «Instalar programas».</li>
        <li><b>Dejá el overlay activo.</b> Al abrirse, abre RTSS y Core Temp en segundo plano si estaban cerrados.</li>
        <li><b>Abrí el juego.</b> RTSS tiene que estar abierto antes de que arranque el juego. El overlay aparece solo, con los FPS y las temperaturas.</li></ol>`,
    },
    {
      t: "Cuándo puede no funcionar",
      art: "",
      body: `<div class="fh-cols"><div><h4>Funciona bien cuando</h4><ul><li>RTSS y Core Temp están abiertos antes de arrancar el juego.</li><li>El juego acepta a RTSS.</li></ul></div>
             <div><h4>Puede fallar cuando</h4><ul><li>RTSS está cerrado o se abrió con el juego ya en marcha: reiniciá el juego.</li><li>Core Temp no está instalado: el procesador no muestra temperatura.</li><li>El anti-cheat de un juego no acepta a RTSS. Eso es cosa de RTSS, no de OverdriveTuner: probalo primero en una práctica, no en una partida con ranking.</li></ul></div></div>`,
    },
  ];
  let i = 0;
  const g = document.createElement("div");
  g.id = "fhGuide";
  g.setAttribute("role", "dialog"); g.setAttribute("aria-modal", "true"); g.setAttribute("aria-label", "Cómo se miden los FPS");
  document.body.appendChild(g);
  const prev = document.activeElement;
  const close = () => { document.removeEventListener("keydown", key, true); g.remove(); if (prev && prev.focus) try { prev.focus(); } catch (e) { /* sin foco */ } };
  const key = e => {
    if (e.key === "Escape") { e.stopPropagation(); close(); }
    else if (e.key === "ArrowRight" && i < scenes.length - 1) { i++; draw(); }
    else if (e.key === "ArrowLeft" && i > 0) { i--; draw(); }
  };
  document.addEventListener("keydown", key, true);
  function draw() {
    const sc = scenes[i];
    g.innerHTML = `<div class="gbox fh-box-main"><div class="fh-pg" aria-hidden="true">${scenes.map((_, k) => `<i class="${k === i ? "on" : k < i ? "done" : ""}"></i>`).join("")}</div>
      <h3>${esc(sc.t)}</h3>${sc.art ? `<div class="fh-art">${sc.art}</div>` : ""}<div class="fh-body">${sc.body}</div>
      <div class="btns" style="margin-top:18px"><button class="btn" id="fhPrev" ${i === 0 ? "disabled" : ""}>Anterior</button>
      ${i < scenes.length - 1 ? '<button class="btn primary" id="fhNext">Siguiente</button>' : '<button class="btn primary" id="fhEnd">Entendido</button>'}
      <button class="btn" id="fhClose" style="margin-left:auto">Cerrar</button></div></div>`;
    g.querySelector("#fhPrev").onclick = () => { i--; draw(); };
    const nx = g.querySelector("#fhNext"); if (nx) nx.onclick = () => { i++; draw(); };
    const en = g.querySelector("#fhEnd"); if (en) en.onclick = close;
    g.querySelector("#fhClose").onclick = close;
    (g.querySelector("#fhNext") || g.querySelector("#fhEnd")).focus();
  }
  draw();
}
