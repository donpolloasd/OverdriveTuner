"use strict";
/* Animación de apertura con sonido generado (WebAudio, sin archivos). Saltable con cualquier tecla o clic. */
(function () {
  const get = k => { try { return localStorage.getItem(k); } catch (e) { return null; } };
  const reduce = window.matchMedia && matchMedia("(prefers-reduced-motion: reduce)").matches;
  const DUR = 4300;

  function sound(vol) {
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC || vol <= 0) return null;
    let ctx;
    try { ctx = new AC(); } catch (e) { return null; }
    const t0 = ctx.currentTime + 0.05;
    const master = ctx.createGain(); master.gain.value = 0;
    const comp = ctx.createDynamicsCompressor();
    master.connect(comp); comp.connect(ctx.destination);
    const level = Math.min(1, vol / 100) * 0.55;
    master.gain.setValueAtTime(0, t0);
    master.gain.linearRampToValueAtTime(level, t0 + 0.4);
    // eco suave
    const dly = ctx.createDelay(1); dly.delayTime.value = 0.27;
    const fb = ctx.createGain(); fb.gain.value = 0.32; const wet = ctx.createGain(); wet.gain.value = 0.35;
    dly.connect(fb); fb.connect(dly); dly.connect(wet); wet.connect(master);
    const bus = ctx.createGain(); bus.connect(master); bus.connect(dly);

    const tone = (freq, start, dur, peak, type, glideTo) => {
      const o = ctx.createOscillator(), g = ctx.createGain();
      o.type = type || "sine"; o.frequency.setValueAtTime(freq, t0 + start);
      if (glideTo) o.frequency.exponentialRampToValueAtTime(glideTo, t0 + start + dur);
      g.gain.setValueAtTime(0.0001, t0 + start);
      g.gain.exponentialRampToValueAtTime(peak, t0 + start + dur * 0.35);
      g.gain.exponentialRampToValueAtTime(0.0001, t0 + start + dur);
      o.connect(g); g.connect(bus); o.start(t0 + start); o.stop(t0 + start + dur + 0.05);
    };
    // oleaje grave
    tone(55, 0, 2.6, 0.5, "sine", 110);
    // acorde de colchón
    [220, 330, 440, 660].forEach((f, i) => tone(f, 0.5 + i * 0.12, 3.0, 0.07 - i * 0.008, "triangle"));
    // campanitas cuando aparece el logo
    tone(1320, 1.5, 1.8, 0.09, "sine"); tone(1760, 1.7, 1.8, 0.06, "sine");
    // soplo filtrado
    const len = Math.floor(ctx.sampleRate * 2.2), buf = ctx.createBuffer(1, len, ctx.sampleRate), ch = buf.getChannelData(0);
    for (let i = 0; i < len; i++) ch[i] = Math.random() * 2 - 1;
    const src = ctx.createBufferSource(); src.buffer = buf;
    const bp = ctx.createBiquadFilter(); bp.type = "bandpass"; bp.Q.value = 0.8;
    bp.frequency.setValueAtTime(300, t0 + 0.2); bp.frequency.exponentialRampToValueAtTime(3200, t0 + 1.5);
    const ng = ctx.createGain(); ng.gain.setValueAtTime(0.0001, t0 + 0.2);
    ng.gain.exponentialRampToValueAtTime(0.1, t0 + 1.2); ng.gain.exponentialRampToValueAtTime(0.0001, t0 + 2.3);
    src.connect(bp); bp.connect(ng); ng.connect(bus); src.start(t0 + 0.2);
    if (ctx.state === "suspended") ctx.resume().catch(() => { });
    return {
      resume() { if (ctx.state === "suspended") ctx.resume().catch(() => { }); },
      stop() { try { master.gain.cancelScheduledValues(ctx.currentTime); master.gain.setValueAtTime(master.gain.value, ctx.currentTime); master.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.2); setTimeout(() => ctx.close().catch(() => { }), 400); } catch (e) { /* ya cerrado */ } },
    };
  }

  /* Devuelve una promesa que se cumple cuando la animación termina o se salta. */
  function play(opts) {
    opts = opts || {};
    const vol = opts.vol == null ? 70 : opts.vol;
    return new Promise(resolve => {
      const root = document.createElement("div");
      root.id = "intro"; root.setAttribute("aria-hidden", "true");
      root.innerHTML = '<canvas></canvas><div class="ilogo"><img src="/logo.png" alt="" width="96" height="96"><b>OverdriveTuner</b></div><small class="iskip">Apretá cualquier tecla para saltar</small>';
      document.body.appendChild(root);
      const cv = root.querySelector("canvas"), g = cv.getContext("2d");
      let W = 0, H = 0, dpr = 1;
      const fit = () => { dpr = Math.min(2, window.devicePixelRatio || 1); W = innerWidth; H = innerHeight; cv.width = W * dpr; cv.height = H * dpr; cv.style.width = W + "px"; cv.style.height = H + "px"; g.setTransform(dpr, 0, 0, dpr, 0, 0); };
      fit(); addEventListener("resize", fit);
      const N = reduce ? 0 : 110;
      const ps = Array.from({ length: N }, () => ({ x: Math.random(), y: Math.random(), r: 0.6 + Math.random() * 2.2, s: 0.02 + Math.random() * 0.06, ph: Math.random() * 6.28, a: 0.15 + Math.random() * 0.5 }));
      const aud = sound(vol);
      const start = performance.now();
      let done = false, raf = 0;
      const ease = t => t < 0 ? 0 : t > 1 ? 1 : 1 - Math.pow(1 - t, 3);
      const frame = now => {
        const t = now - start, k = t / DUR;
        g.clearRect(0, 0, W, H);
        g.fillStyle = "#000"; g.fillRect(0, 0, W, H);
        const cx = W / 2, cy = H / 2, m = Math.min(W, H);
        // niebla de luz que respira
        const gl = ease(k * 1.6) * (1 - ease((k - 0.8) * 5));
        const rg = g.createRadialGradient(cx, cy, 0, cx, cy, m * (0.35 + 0.1 * Math.sin(t / 700)));
        rg.addColorStop(0, `rgba(235,240,255,${0.28 * gl})`); rg.addColorStop(0.5, `rgba(120,140,190,${0.12 * gl})`); rg.addColorStop(1, "rgba(0,0,0,0)");
        g.fillStyle = rg; g.fillRect(0, 0, W, H);
        // partículas flotando
        g.globalCompositeOperation = "lighter";
        for (const p of ps) {
          const y = ((p.y - t * 0.00004 * (1 + p.s * 10)) % 1 + 1) % 1, x = p.x + Math.sin(t / 1300 + p.ph) * 0.02;
          const a = p.a * ease(k * 2) * (1 - ease((k - 0.85) * 6));
          g.fillStyle = `rgba(220,230,255,${a})`; g.beginPath(); g.arc(x * W, y * H, p.r, 0, 6.283); g.fill();
        }
        // anillo que se expande al aparecer el logo
        const rk = (t - 1300) / 1500;
        if (rk > 0 && rk < 1) { g.strokeStyle = `rgba(255,255,255,${0.5 * (1 - rk)})`; g.lineWidth = 2; g.beginPath(); g.arc(cx, cy, m * 0.08 + ease(rk) * m * 0.42, 0, 6.283); g.stroke(); }
        g.globalCompositeOperation = "source-over";
        const logo = root.querySelector(".ilogo");
        const lo = ease((t - 1100) / 900) * (1 - ease((t - (DUR - 700)) / 600));
        logo.style.opacity = lo; logo.style.transform = `scale(${0.92 + 0.08 * ease((t - 1100) / 1400)})`;
        logo.style.filter = `drop-shadow(0 0 ${22 * lo}px rgba(255,255,255,${0.55 * lo}))`;
        if (t >= DUR) return finish();
        raf = requestAnimationFrame(frame);
      };
      const finish = () => {
        if (done) return; done = true;
        cancelAnimationFrame(raf); removeEventListener("keydown", skip, true); removeEventListener("pointerdown", skip, true); removeEventListener("resize", fit);
        if (aud) aud.stop();
        root.classList.add("out");
        setTimeout(() => { root.remove(); resolve(); }, 380);
      };
      const skip = e => { if (e && e.type === "keydown" && (e.key === "Shift" || e.key === "Control" || e.key === "Alt")) return; if (e && e.cancelable) e.preventDefault(); if (aud) aud.resume(); finish(); };
      addEventListener("keydown", skip, true); addEventListener("pointerdown", skip, true);
      if (aud) { const r = () => aud.resume(); addEventListener("pointerdown", r, { once: true }); }
      raf = requestAnimationFrame(frame);
    });
  }

  window.OTIntro = { play, DUR };
  window.introActive = false;
  const B = window.BOOT || {};
  const off = B.introOff !== undefined ? !!B.introOff : get("otIntroOff") === "1";
  if (!off && !new URLSearchParams(location.search).has("nointro")) {
    window.introActive = true;
    const vol = Number.isFinite(+B.introVol) && B.introVol !== undefined ? +B.introVol : (get("otIntroVol") == null ? 70 : +get("otIntroVol"));
    const run = () => play({ vol }).then(() => { window.introActive = false; if (window.introReady) window.introReady(); });
    if (document.body) run(); else addEventListener("DOMContentLoaded", run);
  }
})();
