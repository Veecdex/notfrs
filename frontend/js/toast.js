// Renders small pill toasts into a #toast-stack container.
// Usage: showToast("Note saved", "success" | "error")
function showToast(message, type = "success") {
  const stack = document.getElementById("toast-stack");
  if (!stack) return;

  const toast = document.createElement("div");
  toast.className = `toast ${type}`;
  const icon = type === "error" ? Icons.alert : Icons.check;
  toast.innerHTML = `${icon}<span>${message}</span>`;
  stack.appendChild(toast);

  requestAnimationFrame(() => toast.classList.add("show"));

  setTimeout(() => {
    toast.classList.remove("show");
    setTimeout(() => toast.remove(), 250);
  }, 2600);
}
