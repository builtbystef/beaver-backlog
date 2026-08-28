// Liveness on the page: about once a second the page asks /changed whether the
// store still looks the way it did, and re-fetches its own view when it does
// not. No payload travels with the answer, so what lands on screen is a fresh
// render of the same address, keeping the URL's filters, the column shown in
// full, and the issue being read, rather than a patch that could disagree with
// the files.
//
// The asking is a short poll, deliberately not a held stream: a browser allows
// only about six plain-HTTP connections per origin, and a per-tab event stream
// starved every click and drag once six tabs were open (rpliqf). A poll is
// over in a millisecond and holds nothing.
//
// Only a view marked data-live redraws. A form is never marked, because a page
// somebody is typing into belongs to them until they submit it.

const view = () => document.getElementById("view");

// pending remembers a change that arrived while the view could not be replaced,
// so the redraw happens as soon as it can rather than being lost.
let pending = false;

// pollMs matches the freshness the retired server-side poll already bounded:
// about a second is under the threshold where a reader would reach for the
// reload button, and cheap enough to leave running all day.
const pollMs = 1000;

// validator is the store as this page last saw it, in the form /changed hands
// out (an ETag). It is null until the first answer establishes the baseline, which
// is the store as it stood when the page was opened. It survives an outage on
// purpose: the comparison is against files, not against a connection, so the
// first answer after a gap says precisely whether anything was missed.
let validator = null;

// asking keeps the polls from overlapping when an answer is slow; the next
// tick asks again anyway.
let asking = false;

async function ask() {
  if (asking || document.hidden) return;
  asking = true;
  try {
    const res = await fetch("/changed", {
      cache: "no-store",
      headers: validator === null ? {} : { "If-None-Match": validator },
    });
    if (res.status === 304) {
      status(false);
      return;
    }
    if (!res.ok) {
      disconnected();
      return;
    }
    const fresh = res.headers.get("ETag");
    const behind = validator !== null && fresh !== validator;
    validator = fresh;
    status(false);
    if (behind) refresh();
  } catch {
    disconnected();
  } finally {
    asking = false;
  }
}

setInterval(ask, pollMs);
ask();

// A hidden tab does not poll, since nobody is looking, so coming back asks
// immediately rather than waiting out the tick.
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) ask();
});

// The store's health, said only on a page the poll can redraw: a form is not
// live, so "disconnected" would be a warning about nothing.
function disconnected() {
  if (view()?.dataset.live !== undefined) status(true);
}

function status(shown) {
  const box = document.getElementById("live-status");
  if (box) box.hidden = !shown;
}

// A drag is the one gesture the board holds state in that markup would take
// with it: the card in hand belongs to the DOM being replaced. The refresh
// waits for the drop, which redraws the board itself anyway. The timeout lets
// drag.js clear the mark first, since both listeners answer the same event.
// Dragging the graph's canvas is the same bargain by a different gesture: it
// marks the body the same way and ends on a pointer release rather than a
// dragend, so the redraw waits for the hand to come off the picture.
// A quick view open over the graph is the same bargain by a third gesture, and
// the only one that ends on no pointer event at all, because Escape is a key,
// so the picture says when it lets go.
document.addEventListener("dragend", () => setTimeout(retry, 0));
document.addEventListener("pointerup", () => setTimeout(retry, 0));
document.addEventListener("focusout", () => setTimeout(retry, 0));
document.addEventListener("beaver:release", () => setTimeout(retry, 0));

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
// for. Classes and timestamp text are what the page's own scripts decorate,
// such as hover marks, an earlier flash, or a timestamp reworded as "2 hours
// ago", so comparing them would light up cards nothing happened to.
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
// reader's hands: a card mid-drag, a view held open over the page, or the
// control they are filling in. A link or a button with the focus ring on it is
// none of those, and waiting on one would strand the redraw for good: the
// browser hands focus back to the graph node a quick view was opened from, and
// nothing takes it away again.
function held() {
  if (document.body.dataset.dragging !== undefined) return true;
  if (document.body.dataset.holding !== undefined) return true;
  const focused = document.activeElement;
  return !!focused && filling(focused) && !!view()?.contains(focused);
}

// filling reports whether an element is one a reader puts something into,
// which is what a redraw must not pull out from under them mid-answer.
function filling(el) {
  return el.matches?.("input, textarea, select, [contenteditable]") ?? false;
}

function retry() {
  if (pending) refresh();
}
