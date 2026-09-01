// Small helpers shared across every page. Kept separate from auth.js and
// notes.js because both of those load on different pages, and this is
// needed by all of them (e.g. index.html doesn't load auth.js, but its
// account settings sheet still needs the password toggle).

/** Toggles a password field between hidden and visible text, swapping the eye icon. */
function initPasswordToggle(inputId, toggleId) {
  const input = document.getElementById(inputId);
  const toggle = document.getElementById(toggleId);
  if (!input || !toggle) return;
  toggle.innerHTML = Icons.eye;
  toggle.addEventListener("click", () => {
    const showing = input.type === "text";
    input.type = showing ? "password" : "text";
    toggle.innerHTML = showing ? Icons.eye : Icons.eyeOff;
  });
}
