// Dragging a card between the board's columns, hand-written and dependency-free
// (ADR 0006). The script never moves a card itself: it posts what the drop
// meant and then redraws the board from the server, so what the reader ends up
// looking at is the files' truth rather than an optimistic guess. That is also
// why a refusal needs no undo — the card was never moved, so it is already back
// where it belongs when the message appears.

// dragged is the card currently in hand, and body[data-dragging] is the same
// fact where anything else can see it: the live refresh reads it to keep the
// board from being swapped out from under a drag in progress.
let dragged = null;

// The browser will not scroll the page itself while a drag is in hand, so a
// long column would strand its cards with the other columns off-screen. While
// a card is held, the pointer near the viewport's top or bottom edge scrolls
// the page — faster the closer to the edge it sits. pointerY is where the
// dragover stream last saw the pointer.
const EDGE = 90;
let pointerY = null;
let scrolling = false;

function autoscroll() {
  if (!dragged) { scrolling = false; return; }
  if (pointerY !== null) {
    const topbar = document.querySelector(".topbar");
    const top = (topbar ? topbar.offsetHeight : 0) + EDGE;
    const bottom = window.innerHeight - EDGE;
    if (pointerY < top) window.scrollBy(0, -Math.min(24, (top - pointerY) / 3));
    else if (pointerY > bottom) window.scrollBy(0, Math.min(24, (pointerY - bottom) / 3));
  }
  requestAnimationFrame(autoscroll);
}

document.addEventListener("dragstart", (event) => {
  const card = event.target.closest(".card[draggable]");
  if (!card) return;
  dragged = card;
  document.body.dataset.dragging = card.dataset.issue;
  card.classList.add("dragging");
  // A card is a link, so the browser would otherwise offer its href as the
  // payload; the issue's id is what a column can act on.
  event.dataTransfer.effectAllowed = "move";
  event.dataTransfer.setData("text/plain", card.dataset.issue);
  pointerY = null;
  if (!scrolling) {
    scrolling = true;
    requestAnimationFrame(autoscroll);
  }
});

document.addEventListener("dragend", () => {
  if (dragged) dragged.classList.remove("dragging");
  dragged = null;
  delete document.body.dataset.dragging;
  for (const col of document.querySelectorAll(".column.over")) col.classList.remove("over");
});

document.addEventListener("dragover", (event) => {
  pointerY = event.clientY;
  const column = event.target.closest(".column");
  if (!column || !dragged) return;
  event.preventDefault(); // the default is to refuse the drop
  event.dataTransfer.dropEffect = "move";
  column.classList.add("over");
});

document.addEventListener("dragleave", (event) => {
  const column = event.target.closest(".column");
  if (column && !column.contains(event.relatedTarget)) column.classList.remove("over");
});

document.addEventListener("drop", (event) => {
  const column = event.target.closest(".column");
  const card = dragged;
  if (!column || !card) return;
  event.preventDefault();
  column.classList.remove("over");
  // Dropped where it already was: not a change, so nothing is posted and the
  // issue's file is left exactly as it stands.
  if (column.dataset.column === card.dataset.state) return;
  drop(card, column.dataset.column);
});

// drop asks the server for the move and then redraws. in-progress is the one
// column with a route of its own, because arriving there also claims the issue.
async function drop(card, state) {
  const claiming = state === "in-progress";
  const url = claiming ? card.dataset.startUrl : card.dataset.stateUrl;
  const body = claiming ? null : new URLSearchParams({ state });
  card.classList.add("pending");
  try {
    const res = await fetch(url, { method: "POST", body, headers: { "X-Requested-With": "drag" } });
    const text = await res.text();
    if (!res.ok) {
      say(message(text) || "The move was refused.");
      return;
    }
    say("");
    await redraw();
  } catch (err) {
    say(`The move could not be sent: ${err}`);
  } finally {
    card.classList.remove("pending");
  }
}

// redraw replaces the columns with a freshly rendered board. It re-fetches the
// address the reader is actually on rather than reusing the redirect's page, so
// whatever the query string asks for — a terminal column shown in full — still
// holds after a drop.
async function redraw() {
  const res = await fetch(window.location.href, { headers: { "X-Requested-With": "drag" } });
  const board = parse(await res.text()).querySelector(".board");
  if (board) document.querySelector(".board").replaceWith(board);
}

// message digs the core's own words out of a refusal page — the server renders
// one page for a refused drop whether or not this script is there to read it.
function message(html) {
  const shown = parse(html).querySelector(".error");
  return shown ? shown.textContent.trim() : "";
}

function parse(html) {
  return new DOMParser().parseFromString(html, "text/html");
}

// say shows what the server refused, beside the board the card snapped back to.
function say(text) {
  const box = document.querySelector(".drop-error");
  if (!box) return;
  box.textContent = text;
  box.hidden = text === "";
}
