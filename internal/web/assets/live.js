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

new EventSource("/events").addEventListener("changed", refresh);

// A drag is the one gesture the board holds state in that markup would take
// with it: the card in hand belongs to the DOM being replaced. The refresh
// waits for the drop, which redraws the board itself anyway. The timeout lets
// drag.js clear the mark first — both listeners answer the same event.
document.addEventListener("dragend", () => setTimeout(retry, 0));
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
  target.innerHTML = await res.text();
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
