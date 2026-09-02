// Captures the web UI at 1440x900 and 2x through the DevTools protocol, with
// the palette chosen by emulating prefers-color-scheme, so the sidebar control
// honestly shows "System" and no browser state leaks into the picture.
//
// usage: node capture.mjs shots.json <outdir>
// spec:  [{ name, url, theme: "light"|"dark", script?, wait? }]
//   script runs in the page after load (the graph uses it to zoom in);
//   wait is the settle time in ms before the capture (default 400).
//
// The doctor view is only worth a picture when there is something to report:
// before capturing it, drift one filename and drop in a file with an invalid
// state, then restore the store. Run the output through quantize.py to bring
// each PNG under the source budget the site tests hold them to.
import { spawn } from 'node:child_process';
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const [spec, out] = process.argv.slice(2);
const shots = JSON.parse(readFileSync(spec, 'utf8'));
mkdirSync(out, { recursive: true });
const WIDTH = 1440, HEIGHT = 900, SCALE = 2;

const profile = mkdtempSync(join(tmpdir(), 'beaver-shots-'));
const chrome = spawn('google-chrome', [
  '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
  '--no-first-run', '--hide-scrollbars', `--window-size=${WIDTH},${HEIGHT}`,
  '--font-render-hinting=none', 'about:blank',
], { stdio: ['ignore', 'ignore', 'pipe'] });

const browserWs = await new Promise((resolve, reject) => {
  let buf = '';
  chrome.stderr.on('data', (d) => {
    buf += d;
    const m = buf.match(/DevTools listening on (ws:\/\/\S+)/);
    if (m) resolve(m[1]);
  });
  chrome.on('exit', (c) => reject(new Error(`chrome exited ${c}\n${buf}`)));
});
const port = new URL(browserWs).port;

class Cdp {
  #ws; #id = 0; #pending = new Map(); #listeners = [];
  constructor(ws) { this.#ws = ws;
    ws.addEventListener('message', (e) => {
      const msg = JSON.parse(e.data);
      if (msg.id) { const p = this.#pending.get(msg.id); this.#pending.delete(msg.id);
        msg.error ? p.reject(new Error(msg.error.message)) : p.resolve(msg.result); }
      else for (const l of this.#listeners) l(msg);
    });
  }
  static async open(url) { const ws = new WebSocket(url); await new Promise((r) => ws.addEventListener('open', r)); return new Cdp(ws); }
  send(method, params = {}) { const id = ++this.#id; this.#ws.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => this.#pending.set(id, { resolve, reject })); }
  on(fn) { this.#listeners.push(fn); }
  once(method) { return new Promise((r) => this.on((m) => m.method === method && r(m.params))); }
  close() { this.#ws.close(); }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const browser = await Cdp.open(browserWs);

for (const shot of shots) {
  const { targetId } = await browser.send('Target.createTarget', { url: 'about:blank' });
  const page = await Cdp.open(`ws://127.0.0.1:${port}/devtools/page/${targetId}`);
  await page.send('Page.enable');
  await page.send('Runtime.enable');
  await page.send('Emulation.setDeviceMetricsOverride', { width: WIDTH, height: HEIGHT, deviceScaleFactor: SCALE, mobile: false });
  await page.send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: shot.theme }] });
  const loaded = page.once('Page.loadEventFired');
  await page.send('Page.navigate', { url: shot.url });
  await loaded;
  await page.send('Runtime.evaluate', { expression: 'document.fonts.ready', awaitPromise: true });
  if (shot.script) {
    const r = await page.send('Runtime.evaluate', { expression: `(async () => { ${shot.script} })()`, awaitPromise: true });
    if (r.exceptionDetails) throw new Error(`${shot.name}: ${JSON.stringify(r.exceptionDetails)}`);
  }
  await sleep(shot.wait ?? 400);
  const { data } = await page.send('Page.captureScreenshot', { format: 'png' });
  const file = join(out, `${shot.name}-${shot.theme}.png`);
  writeFileSync(file, Buffer.from(data, 'base64'));
  console.log(file);
  page.close();
  await browser.send('Target.closeTarget', { targetId });
}
browser.close();
chrome.kill();
