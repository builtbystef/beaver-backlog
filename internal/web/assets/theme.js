// The reader's say over which palette the UI draws in: system, which is no
// override at all, light, or dark. The choice is one browser's, never the
// store's, so it lives in localStorage and is posted nowhere.
//
// This is the one script the shell does not defer. The palette has to be in
// force at the first paint; a deferred script runs after the document is
// parsed, which is a page drawn in one palette and swapped to the other in
// front of the reader.

const themeKey = "beaver.theme";

// applyTheme writes the override onto the root element, where the stylesheet's
// palette blocks read it. system carries no attribute rather than a name of its
// own: an absent override is what hands the palette back to the operating
// system's preference.
function applyTheme(choice) {
  if (choice === "light" || choice === "dark") {
    document.documentElement.dataset.theme = choice;
  } else {
    delete document.documentElement.dataset.theme;
  }
}

// storedTheme is the state this browser remembers, and system when it remembers
// none. A browser that refuses storage (a private window, or site data turned
// off) is answered the same way: there is nothing to fall back to but the
// operating system.
function storedTheme() {
  try {
    return localStorage.getItem(themeKey) || "system";
  } catch {
    return "system";
  }
}

applyTheme(storedTheme());

// The control is rendered on system, that being what the server can honestly
// say; the state this browser remembers is put back the moment the control
// exists.
document.addEventListener("DOMContentLoaded", () => {
  const state = document.querySelector(`#theme input[value="${storedTheme()}"]`);
  if (state) state.checked = true;
});

// swapTheme is applyTheme for a palette the reader just picked, which is the
// one time the change is watched. Every colour on the page moves at once, and
// the elements carrying a colour transition for their hover would ease to the
// new palette while everything without one has already arrived, so the
// transitions are off for the swap.
function swapTheme(choice) {
  const root = document.documentElement;
  root.classList.add("theme-switching");
  applyTheme(choice);
  // Reading a layout property forces the recalculation to happen here, with the
  // transitions still off, so the new palette is already what they are
  // measured from when they come back on the next line.
  void root.offsetHeight;
  root.classList.remove("theme-switching");
}

document.addEventListener("change", (event) => {
  const state = event.target;
  if (!state.closest?.("#theme")) return;
  swapTheme(state.value);
  try {
    if (state.value === "system") localStorage.removeItem(themeKey);
    else localStorage.setItem(themeKey, state.value);
  } catch {
    // A browser refusing storage still gets the palette it was just asked for;
    // it simply will not remember it past this page.
  }
});
