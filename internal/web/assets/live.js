// Liveness on the page: the server says the store changed, and the view
// re-fetches itself. No payload travels with the event, so what lands on screen
// is a fresh render of the same address — the URL's filters, the column shown
// in full, the issue being read — rather than a patch that could disagree with
// the files.
//
// Only a view marked data-live redraws. A form is never marked, because a page
// somebody is typing into belongs to them until they submit it.

const view = () => document.getElementById("view");

// pending remembers a change that arrived while the view could not be replaced,
// so the redraw happens as soon as it can rather than being lost.
let pending = false;

const feed = new EventSource("/events");
feed.addEventListener("changed", refresh);

// The feed's health, said only on a page the feed can redraw: a form is not
// live, so "disconnected" would be a warning about nothing. dropped remembers
// an outage so the reconnect can ask what was missed — the events themselves
// carry no payload, so anything announced during the gap is simply gone.
let dropped = false;

feed.addEventListener("error", () => {
  dropped = true;
  if (view()?.dataset.live !== undefined) status(true);
});

feed.addEventListener("open", () => {
  status(false);
  if (!dropped) return;
  dropped = false;
  refresh();
});

function status(shown) {
  const box = document.getElementById("live-status");
  if (box) box.hidden = !shown;
}

// A drag is the one gesture the board holds state in that markup would take
// with it: the card in hand belongs to the DOM being replaced. The refresh
// waits for the drop, which redraws the board itself anyway. The timeout lets
// drag.js clear the mark first — both listeners answer the same event.
// Dragging the graph's canvas is the same bargain by a different gesture: it
// marks the body the same way and ends on a pointer release rather than a
// dragend, so the redraw waits for the hand to come off the picture.
document.addEventListener("dragend", () => setTimeout(retry, 0));
document.addEventListener("pointerup", () => setTimeout(retry, 0));
document.addEventListener("focusout", () => setTimeout(retry, 0));

async function refresh() {
  const target = view();
  if (!target || target.dataset.live === undefined) return;
  if (held()) {
    pending = true;
    return;
  }
  pending = false;
  const res = await fetch(window.location.href, { headers: { "HX-Request": "true" } });
  if (!res.ok) return; // the store may be mid-checkout; the next change asks again
  const before = snapshot(target);
  target.innerHTML = await res.text();
  // What the redraw changed gets a moment of light: the reader did not ask for
  // this render, so the page says what moved rather than moving silently.
  for (const el of view()?.querySelectorAll("[data-issue]") ?? []) {
    if (before.get(el.dataset.issue) !== fingerprint(el)) el.classList.add("changed");
  }
}

// snapshot is each issue-bearing element's fingerprint before a redraw, keyed
// by the issue it shows.
function snapshot(target) {
  const before = new Map();
  for (const el of target.querySelectorAll("[data-issue]")) before.set(el.dataset.issue, fingerprint(el));
  return before;
}

// fingerprint is the part of an element's markup the server is responsible
// for. Classes and timestamp text are what the page's own scripts decorate —
// hover marks, an earlier flash, a timestamp reworded as "2 hours ago" — so
// comparing them would light up cards nothing happened to.
function fingerprint(el) {
  const clone = el.cloneNode(true);
  for (const each of [clone, ...clone.querySelectorAll("[class]")]) each.removeAttribute("class");
  for (const time of clone.querySelectorAll("time")) {
    time.textContent = time.getAttribute("datetime");
    time.removeAttribute("title");
  }
  return clone.outerHTML;
}

// held reports whether redrawing right now would take something out of the
// reader's hands: a card mid-drag, or the field they are typing in.
function held() {
  if (document.body.dataset.dragging !== undefined) return true;
  const focused = document.activeElement;
  return !!focused && focused !== document.body && !!view()?.contains(focused);
}

function retry() {
  if (pending) refresh();
}
