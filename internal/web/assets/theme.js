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

document.addEventListener("change", (event) => {
  const state = event.target;
  if (!state.closest?.("#theme")) return;
  applyTheme(state.value);
  try {
    if (state.value === "system") localStorage.removeItem(themeKey);
    else localStorage.setItem(themeKey, state.value);
  } catch {
    // A browser refusing storage still gets the palette it was just asked for;
    // it simply will not remember it past this page.
  }
});
