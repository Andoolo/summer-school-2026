// Мини-клиент Chrome DevTools Protocol: одна команда за запуск, Chrome живёт отдельно.
// node cdp.mjs goto <url> | eval "<js>" | scroll <x> <y> <dy> | shot <file.png>
// Экран — как у телефона: 390×844 @3x. Нужен Node 22+ (встроенный WebSocket).
import { writeFileSync } from "node:fs";

const PORT = Number(process.env.CDP_PORT || 9333);
const [, , cmd, ...args] = process.argv;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function page() {
  const list = await (await fetch(`http://127.0.0.1:${PORT}/json/list`)).json();
  let target = list.find((t) => t.type === "page");
  if (!target) target = await (await fetch(`http://127.0.0.1:${PORT}/json/new?about:blank`, { method: "PUT" })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((ok, fail) => { ws.onopen = ok; ws.onerror = fail; });
  let id = 0;
  const pending = new Map();
  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.id && pending.has(msg.id)) {
      const { ok, fail } = pending.get(msg.id);
      pending.delete(msg.id);
      msg.error ? fail(new Error(JSON.stringify(msg.error))) : ok(msg.result);
    }
  };
  const send = (method, params = {}) => new Promise((ok, fail) => {
    const n = ++id;
    pending.set(n, { ok, fail });
    ws.send(JSON.stringify({ id: n, method, params }));
  });
  return { send, close: () => ws.close() };
}

async function evaluate(p, expression) {
  const res = await p.send("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
  if (res.exceptionDetails) throw new Error(res.exceptionDetails.exception?.description || "eval failed");
  return res.result.value;
}

// Приложение готово, когда index.html спрятал экран загрузки. В CI рендер программный,
// поэтому ждём до 90 секунд.
async function waitReady(p) {
  for (let i = 0; i < 180; i++) {
    const ready = await evaluate(p, `(() => { const o = document.getElementById('loading-overlay'); return !o || o.classList.contains('hidden') || getComputedStyle(o).opacity === '0'; })()`).catch(() => false);
    if (ready) return;
    await sleep(500);
  }
  throw new Error("app did not become ready");
}

const p = await page();
try {
  await p.send("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 3, mobile: true });
  if (cmd === "goto") {
    await p.send("Page.enable");
    await p.send("Page.navigate", { url: args[0] });
    await sleep(1500);
    await waitReady(p);
    await sleep(2500);
    console.log("ready", await evaluate(p, "location.href"));
  } else if (cmd === "eval") {
    console.log(JSON.stringify(await evaluate(p, args[0])));
  } else if (cmd === "scroll") {
    const [x, y, dy] = args.map(Number);
    await p.send("Input.dispatchMouseEvent", { type: "mouseWheel", x, y, deltaX: 0, deltaY: dy });
    await sleep(1500);
    console.log("scrolled", dy);
  } else if (cmd === "shot") {
    await sleep(800);
    const { data } = await p.send("Page.captureScreenshot", { format: "png" });
    writeFileSync(args[0], Buffer.from(data, "base64"));
    console.log("saved", args[0]);
  } else {
    throw new Error("unknown command " + cmd);
  }
} finally {
  p.close();
}
