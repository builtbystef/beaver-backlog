// The graph as a canvas: panning, zooming and hover-highlighting, hand-written
// and dependency-free (ADR 0006). Nothing here draws anything. The picture is
// the server's, already laid out, so the script only moves the window onto it
// and marks what the pointer is near.
//
// The window is the SVG's viewBox, in the same user units the layout was
// computed in: panning subtracts the pointer's travel from its corner, zooming
// shrinks it about the point under the cursor, resetting returns it to the whole
// picture. Holding it here rather than in the markup is what lets a live redraw
// land under a reader without moving the ground: the fresh picture is handed the
// window the old one had.

// viewport is the part of the picture on screen, in user units. It outlives
// every redraw of the view, and is null only before the first picture is fitted.
let viewport = null;

// panning is the gesture in progress: where the pointer was last seen, and how
// many user units a screen pixel covered when it went down.
let panning = null;

const canvas = () => document.querySelector("svg.graph");

document.addEventListener("DOMContentLoaded", adopt);

// A redraw, whether the live listener's or a filter's htmx swap, replaces the
// picture with a fresh one that knows nothing of where the reader had panned to.
// The observer is how this script hears about that without either of them
// having to know it is here.
new MutationObserver(adopt).observe(document.documentElement, { childList: true, subtree: true });

// adopt turns a freshly rendered picture into the one being read: the window the
// reader had is put back over it, so a redraw lands underneath them without
// moving the ground.
function adopt() {
  const svg = canvas();
  if (!svg || svg.dataset.adopted !== undefined) return;
  svg.dataset.adopted = "";
  if (!viewport) viewport = whole(svg);
  show(svg);
}

// whole is the window holding the entire picture, which is also what reset
// returns to. Read from the width and height the server rendered, never from the
// viewBox, which show() keeps overwriting with wherever the reader has panned.
function whole(svg) {
  return { x: 0, y: 0, w: Number(svg.getAttribute("width")), h: Number(svg.getAttribute("height")) };
}

function show(svg) {
  svg.setAttribute("viewBox", `${viewport.x} ${viewport.y} ${viewport.w} ${viewport.h}`);
  ground(svg);
}

// The dotted canvas is drawn inside the picture rather than painted behind the
// frame, which is what makes it travel with a pan and grow with a zoom. But it
// is a rectangle, and the server can only size it to the layout's own extent, so
// it gets restretched to whatever the window covers. The dots do not move with
// it, the pattern being laid out in the picture's units, not the rectangle's.
//
// A window's width past every edge, because the window is not the whole of what
// is on screen: where the frame and the picture disagree in shape the browser
// letterboxes, and those bands are outside the viewBox but still in plain
// sight.
function ground(svg) {
  const dots = svg.querySelector(".grid");
  if (!dots) return;
  dots.setAttribute("x", viewport.x - viewport.w);
  dots.setAttribute("y", viewport.y - viewport.h);
  dots.setAttribute("width", viewport.w * 3);
  dots.setAttribute("height", viewport.h * 3);
}

document.addEventListener("click", (event) => {
  const svg = canvas();
  if (!svg || !event.target.closest("[data-graph-reset]")) return;
  viewport = whole(svg);
  show(svg);
});

// The zoom buttons are the wheel's gesture for a hand without one: the same
// clamped step, about the middle of the window rather than a cursor.
document.addEventListener("click", (event) => {
  const svg = canvas();
  const control = event.target.closest("[data-graph-zoom]");
  if (!svg || !viewport || !control) return;
  zoom(svg, control.dataset.graphZoom === "in" ? 1 / 1.3 : 1.3, {
    x: viewport.x + viewport.w / 2,
    y: viewport.y + viewport.h / 2,
  });
});

// zoom resizes the window by wanted (clamped to the same bounds however it was
// asked for), keeping the point at about where it is.
function zoom(svg, wanted, at) {
  const full = whole(svg);
  const factor = clamp(viewport.w * wanted, full.w / 40, full.w * 4) / viewport.w;
  viewport = {
    x: at.x - (at.x - viewport.x) * factor,
    y: at.y - (at.y - viewport.y) * factor,
    w: viewport.w * factor,
    h: viewport.h * factor,
  };
  show(svg);
}

// Panning drags the background only: a node is a link, and dragging one would
// cost the reader the click that follows it.
document.addEventListener("pointerdown", (event) => {
  const svg = canvas();
  if (!svg || event.button !== 0 || !svg.contains(event.target) || event.target.closest("[data-issue]")) return;
  panning = { x: event.clientX, y: event.clientY, scale: perPixel(svg) };
  svg.closest(".graph-frame")?.classList.add("panning");
  // The live refresh reads this the way it reads a card mid-drag: the picture is
  // never swapped out from under a hand that is moving it.
  document.body.dataset.dragging = "graph";
  svg.setPointerCapture?.(event.pointerId);
});

document.addEventListener("pointermove", (event) => {
  const svg = canvas();
  if (!panning || !svg) return;
  event.preventDefault();
  // The travel is measured in screen pixels and spent in user units, at the
  // scale the gesture began at, so the picture keeps tracking the pointer
  // however far it has already been dragged.
  viewport = {
    ...viewport,
    x: viewport.x - (event.clientX - panning.x) * panning.scale,
    y: viewport.y - (event.clientY - panning.y) * panning.scale,
  };
  panning.x = event.clientX;
  panning.y = event.clientY;
  show(svg);
});

for (const ending of ["pointerup", "pointercancel"]) {
  document.addEventListener(ending, () => {
    if (!panning) return;
    panning = null;
    canvas()?.closest(".graph-frame")?.classList.remove("panning");
    delete document.body.dataset.dragging;
  });
}

// Zooming keeps the point under the cursor where it is: the window grows or
// shrinks about it, so the reader magnifies what they are pointing at rather
// than whatever happens to be in the middle.
document.addEventListener(
  "wheel",
  (event) => {
    const svg = canvas();
    if (!svg || !viewport || !svg.contains(event.target)) return;
    event.preventDefault();
    zoom(svg, Math.exp(event.deltaY * 0.0015), user(svg, event));
  },
  { passive: false },
);

// Hovering a node lifts its neighbourhood out of the picture: the node, the
// arrows at either end of it, and the issues those arrows reach. Everything else
// dims, which is what makes one issue's dependencies readable in a backlog too
// big to follow a line across.
document.addEventListener("pointerover", (event) => {
  const svg = canvas();
  const node = event.target.closest?.("[data-issue]");
  if (!svg || !node || !svg.contains(node)) return;
  const near = new Set([node.dataset.issue]);
  for (const arrow of svg.querySelectorAll("[data-edge]")) {
    const [from, to] = arrow.dataset.edge.split("→");
    if (from !== node.dataset.issue && to !== node.dataset.issue) continue;
    arrow.classList.add("near");
    near.add(from);
    near.add(to);
  }
  for (const other of svg.querySelectorAll("[data-issue]")) {
    if (near.has(other.dataset.issue)) other.classList.add("near");
  }
  svg.classList.add("dimmed");
});

document.addEventListener("pointerout", (event) => {
  const svg = canvas();
  const node = event.target.closest?.("[data-issue]");
  if (!svg || !node) return;
  // A node is several elements (box, title, badges) and crossing between them is
  // not leaving it, so the neighbourhood must not flicker on the way past.
  if (node.contains(event.relatedTarget)) return;
  svg.classList.remove("dimmed");
  for (const marked of svg.querySelectorAll(".near")) marked.classList.remove("near");
});

// user maps a pointer event into the picture's own units, asking the browser for
// the transform rather than deriving it: the frame letterboxes the picture when
// their shapes disagree, and only the browser knows by how much.
function user(svg, event) {
  const inverse = svg.getScreenCTM()?.inverse();
  if (!inverse) return { x: viewport.x + viewport.w / 2, y: viewport.y + viewport.h / 2 };
  return new DOMPoint(event.clientX, event.clientY).matrixTransform(inverse);
}

// perPixel is how many user units one screen pixel currently covers.
function perPixel(svg) {
  const ctm = svg.getScreenCTM();
  return ctm && ctm.a ? 1 / ctm.a : viewport.w / (svg.clientWidth || viewport.w);
}

function clamp(value, low, high) {
  return Math.min(Math.max(value, low), high);
}
