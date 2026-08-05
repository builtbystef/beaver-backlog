// Small comforts over the rendered page, hand-written and dependency-free
// (ADR 0006): timestamps said as distances, table rows that click like the
// links they hold, and note boxes that grow with what is typed into them. Each
// is decoration over markup that already works — without this script the times
// are absolute, the title cell is still a link, and a textarea still scrolls.

// The server writes every timestamp absolute inside <time datetime="…">; this
// rewords it as a distance ("2 hours ago") and keeps the absolute form in the
// tooltip. Re-run on a timer so an open page does not drift stale, and after
// every redraw — htmx's swaps and the live refresh both land inside #view,
// where the observer below is watching.
function reword() {
  for (const time of document.querySelectorAll("time[datetime]")) {
    const at = new Date(time.getAttribute("datetime"));
    if (Number.isNaN(at.getTime())) continue;
    if (!time.title) time.title = time.textContent.trim();
    const words = distance(at);
    // Only a wording that actually changed is written back: the observer below
    // hears every write, and an unconditional one would echo forever.
    if (time.textContent !== words) time.textContent = words;
  }
}

// distance words how long ago (or, for a hand-edited file from the future,
// until) a moment is. The step below a unit's floor keeps the words honest:
// 59 minutes is "59 minutes ago", 61 is "an hour ago".
function distance(at) {
  const seconds = Math.round((Date.now() - at.getTime()) / 1000);
  const ago = seconds >= 0;
  const spans = [
    [60, "just now", null],
    [3600, "a minute", 60],
    [86400, "an hour", 3600],
    [86400 * 30, "a day", 86400],
    [86400 * 365, "a month", 86400 * 30],
    [Infinity, "a year", 86400 * 365],
  ];
  const s = Math.abs(seconds);
  for (const [limit, one, unit] of spans) {
    if (s >= limit) continue;
    if (!unit) return ago ? one : "in a moment";
    const n = Math.floor(s / unit);
    const words = n === 1 ? one : `${n} ${one.replace(/^an? /, "")}s`;
    return ago ? `${words} ago` : `in ${words}`;
  }
}

// A row that names an issue navigates like the card it mirrors. Only a click
// on the row's own background counts — a link, a badge with a link in it, or a
// hand selecting text keeps its own meaning.
document.addEventListener("click", (event) => {
  const row = event.target.closest("tr[data-href]");
  if (!row || event.target.closest("a, button, input, label")) return;
  if (window.getSelection()?.toString()) return;
  window.location.href = row.dataset.href;
});

// A textarea grows to hold what is in it, within reason, so writing a long
// note or description is not done through a letterbox.
document.addEventListener("input", (event) => {
  const box = event.target;
  if (!(box instanceof HTMLTextAreaElement)) return;
  box.style.height = "auto";
  box.style.height = Math.min(box.scrollHeight + 2, window.innerHeight * 0.6) + "px";
});

document.addEventListener("DOMContentLoaded", reword);
setInterval(reword, 60_000);
// Redraws land wholesale — innerHTML from the live listener, swapped nodes
// from htmx — so element-level watching would miss them; watching the tree
// catches both without knowing either is there.
new MutationObserver(reword).observe(document.documentElement, { childList: true, subtree: true });
