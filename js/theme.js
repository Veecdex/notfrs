// Applies the saved theme (or system preference) before paint, and wires
// up any .theme-toggle button on the page.
(function applyStoredTheme() {
  const saved = localStorage.getItem("notes_theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  const theme = saved || (prefersDark ? "dark" : "light");
  document.documentElement.setAttribute("data-theme", theme);
})();

function initThemeToggle(buttonId = "theme-toggle") {
  const btn = document.getElementById(buttonId);
  if (!btn) return;

  btn.addEventListener("click", () => {
    const current = document.documentElement.getAttribute("data-theme");
    const next = current === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    localStorage.setItem("notes_theme", next);
  });
}
