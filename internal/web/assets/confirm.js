// Confirmation for a form that cannot be undone. A form carrying
// data-confirm="<dialog id>" opens that dialog instead of submitting, and
// submits only once the dialog closes with "confirm". Without JavaScript the
// form still posts: a delete behind a broken script is worse than one behind
// no script.
document.addEventListener("submit", (event) => {
  const form = event.target;
  const dialog = document.getElementById(form.dataset.confirm || "");
  if (!dialog || form.dataset.confirmed) return;
  event.preventDefault();
  dialog.addEventListener(
    "close",
    () => {
      if (dialog.returnValue !== "confirm") return;
      form.dataset.confirmed = "true";
      form.requestSubmit();
    },
    { once: true },
  );
  dialog.returnValue = "";
  dialog.showModal();
});
