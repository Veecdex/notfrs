let allNotes = [];
let editingNoteId = null;
let pendingDeleteId = null;
let pendingAvatarDataUrl = null;
const activeBackdropSheets = new Set();

function isDesktop() {
  return window.matchMedia("(min-width: 900px)").matches;
}

function formatTimestamp(isoString) {
  const d = new Date(isoString);
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  if (sameDay) {
    return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  }
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function initials(name, email) {
  const source = (name || "").trim() || (email || "").trim();
  if (!source) return "?";
  const parts = source.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return source.slice(0, 2).toUpperCase();
}

/** Fills any element with class "avatar" with either the photo or initials. */
function renderAvatarInto(el, { name, email, avatarDataUrl }) {
  if (avatarDataUrl) {
    el.innerHTML = `<img src="${avatarDataUrl}" alt="" />`;
  } else {
    el.textContent = initials(name, email);
  }
}

/* ---------- Skeletons ---------- */
function renderSkeletons(count = 5) {
  const list = document.getElementById("notes-list");
  list.innerHTML = "";
  for (let i = 0; i < count; i++) {
    const row = document.createElement("div");
    row.className = "skeleton-row";
    row.innerHTML = `
      <div class="skeleton-dot"></div>
      <div class="skeleton-lines">
        <div class="skeleton-line w70"></div>
        <div class="skeleton-line w40"></div>
      </div>
    `;
    list.appendChild(row);
  }
}

/* ---------- Loading + rendering notes ---------- */
async function loadNotes() {
  renderSkeletons();
  try {
    allNotes = await apiFetch("/api/notes");
    renderNotes(allNotes);
  } catch (err) {
    if (isSessionError(err.message)) {
      logout();
      return;
    }
    document.getElementById("notes-list").innerHTML = "";
    showToast(err.message, "error");
  }
}

function renderNotes(notes) {
  const list = document.getElementById("notes-list");
  const empty = document.getElementById("empty-state");
  list.innerHTML = "";

  if (!notes || notes.length === 0) {
    empty.style.display = "block";
    return;
  }
  empty.style.display = "none";

  for (const note of notes) {
    list.appendChild(renderNoteRow(note));
  }
  highlightSelectedRow();
}

function renderNoteRow(note) {
  const row = document.createElement("div");
  row.className = "note-row";
  row.dataset.id = note.id;
  row.innerHTML = `
    <div class="note-dot"></div>
    <div class="note-row-body">
      <p class="note-row-title">${escapeHtml(note.title)}</p>
      <p class="note-row-preview">${escapeHtml(note.content || "No additional text")}</p>
      <p class="note-row-meta">${formatTimestamp(note.updated_at)}</p>
    </div>
    <button class="note-row-delete" title="Delete note" data-action="delete">${Icons.trash}</button>
  `;

  row.addEventListener("click", (e) => {
    if (e.target.closest('[data-action="delete"]')) return;
    openEditor(note);
  });

  row.querySelector('[data-action="delete"]').addEventListener("click", (e) => {
    e.stopPropagation();
    openDeleteConfirm(note.id);
  });

  return row;
}

function highlightSelectedRow() {
  document.querySelectorAll(".note-row").forEach((row) => {
    row.classList.toggle("active", String(editingNoteId) === row.dataset.id);
  });
}

/* ---------- Search ---------- */
function initSearch() {
  const input = document.getElementById("search-input");
  input.addEventListener("input", () => {
    const q = input.value.trim().toLowerCase();
    if (!q) {
      renderNotes(allNotes);
      return;
    }
    const filtered = allNotes.filter(
      (n) => n.title.toLowerCase().includes(q) || n.content.toLowerCase().includes(q)
    );
    renderNotes(filtered);
  });
}

/* ---------- Editor panel/sheet ---------- */
function openEditor(note = null) {
  editingNoteId = note ? note.id : null;
  document.getElementById("editor-heading").textContent = note ? "Edit note" : "New note";
  document.getElementById("note-title-input").value = note ? note.title : "";
  document.getElementById("note-content-input").value = note ? note.content : "";

  document.getElementById("editor-empty").style.display = "none";
  document.getElementById("note-form").style.display = "flex";

  openSheet("editor-sheet", !isDesktop());
  highlightSelectedRow();
  setTimeout(() => document.getElementById("note-title-input").focus(), isDesktop() ? 0 : 300);
}

function closeEditor() {
  closeSheet("editor-sheet");
  editingNoteId = null;
  highlightSelectedRow();

  if (isDesktop()) {
    document.getElementById("note-form").style.display = "none";
    document.getElementById("editor-empty").style.display = "flex";
  }
}

async function saveNote(e) {
  e.preventDefault();
  const title = document.getElementById("note-title-input").value.trim();
  const content = document.getElementById("note-content-input").value;
  const saveBtn = document.getElementById("save-note-btn");

  if (!title) {
    showToast("Give the note a title first", "error");
    return;
  }

  saveBtn.disabled = true;
  saveBtn.innerHTML = `<span class="spinner"></span> Save`;

  try {
    if (editingNoteId) {
      await apiFetch(`/api/notes/${editingNoteId}`, { method: "PUT", body: { title, content } });
      showToast("Note updated", "success");
    } else {
      await apiFetch("/api/notes", { method: "POST", body: { title, content } });
      showToast("Note saved", "success");
    }
    closeEditor();
    loadNotes();
  } catch (err) {
    showToast(err.message, "error");
  } finally {
    saveBtn.disabled = false;
    saveBtn.innerHTML = "Save";
  }
}

/* ---------- Delete confirm (action sheet / modal) ---------- */
function openDeleteConfirm(noteId) {
  pendingDeleteId = noteId;
  openSheet("delete-sheet");
}

function closeDeleteConfirm() {
  closeSheet("delete-sheet");
  pendingDeleteId = null;
}

async function confirmDelete() {
  if (!pendingDeleteId) return;
  const id = pendingDeleteId;
  closeDeleteConfirm();

  try {
    await apiFetch(`/api/notes/${id}`, { method: "DELETE" });
    showToast("Note deleted", "success");
    if (editingNoteId === id) closeEditor();
    loadNotes();
  } catch (err) {
    showToast(err.message, "error");
  }
}

/* ---------- Generic sheet open/close helpers ---------- */
function openSheet(sheetId, withBackdrop = true) {
  document.getElementById(sheetId).classList.add("open");
  if (withBackdrop) {
    activeBackdropSheets.add(sheetId);
    document.getElementById("sheet-backdrop").classList.add("open");
  }
}

function closeSheet(sheetId) {
  document.getElementById(sheetId).classList.remove("open");
  activeBackdropSheets.delete(sheetId);
  if (activeBackdropSheets.size === 0) {
    document.getElementById("sheet-backdrop").classList.remove("open");
  }
}

function closeAllSheets() {
  document.querySelectorAll(".sheet.open").forEach((s) => {
    // Never let an outside click dismiss the always-visible desktop editor panel.
    if (isDesktop() && s.id === "editor-sheet") return;
    s.classList.remove("open");
  });
  activeBackdropSheets.clear();
  document.getElementById("sheet-backdrop").classList.remove("open");
  pendingDeleteId = null;
}

/* =========================================================
   Account settings
   ========================================================= */
let currentProfile = null;

async function loadProfileIntoUI() {
  try {
    currentProfile = await apiFetch("/api/me");
    Storage.saveProfile(currentProfile.name, currentProfile.avatar_data_url);
    paintIdentity();
  } catch (err) {
    if (isSessionError(err.message)) {
      logout();
      return;
    }
    // Fall back to whatever we cached locally; not fatal.
    paintIdentity();
  }
}

function paintIdentity() {
  const name = currentProfile ? currentProfile.name : Storage.getName();
  const email = currentProfile ? currentProfile.email : Storage.getEmail();
  const avatar = currentProfile ? currentProfile.avatar_data_url : Storage.getAvatar();

  document.getElementById("sidebar-user-name").textContent = name || email;
  document.getElementById("sidebar-user-email").textContent = email || "";
  document.querySelectorAll(".avatar[data-role='identity']").forEach((el) => {
    renderAvatarInto(el, { name, email, avatarDataUrl: avatar });
  });
}

async function openSettings() {
  // Don't silently do nothing if /api/me hasn't resolved yet (e.g. clicked
  // very quickly after page load) - go get it now instead of giving up.
  if (!currentProfile) {
    try {
      currentProfile = await apiFetch("/api/me");
      Storage.saveProfile(currentProfile.name, currentProfile.avatar_data_url);
    } catch (err) {
      if (isSessionError(err.message)) {
        logout();
        return;
      }
      // Still couldn't get a fresh profile - fall back to whatever's
      // cached locally so the sheet can open with something useful
      // rather than staying dead.
      const cachedEmail = Storage.getEmail();
      if (!cachedEmail) {
        showToast(err.message || "Couldn't load your account", "error");
        return;
      }
      currentProfile = {
        name: Storage.getName(),
        email: cachedEmail,
        avatar_data_url: Storage.getAvatar(),
      };
      showToast("Showing your last saved profile - couldn't refresh it just now", "error");
    }
  }

  document.getElementById("settings-name-input").value = currentProfile.name || "";
  document.getElementById("settings-email-input").value = currentProfile.email || "";
  pendingAvatarDataUrl = currentProfile.avatar_data_url || null;
  renderAvatarInto(document.getElementById("settings-avatar-preview"), {
    name: currentProfile.name,
    email: currentProfile.email,
    avatarDataUrl: pendingAvatarDataUrl,
  });
  document.getElementById("password-current").value = "";
  document.getElementById("password-new").value = "";
  document.getElementById("password-confirm").value = "";
  openSheet("settings-sheet");
}

function closeSettings() {
  closeSheet("settings-sheet");
}

/** Resizes an uploaded image client-side so we're not storing multi-megabyte avatars as base64 text. */
function resizeImageFile(file, maxSize = 240) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error("Couldn't read that image"));
    reader.onload = () => {
      const img = new Image();
      img.onerror = () => reject(new Error("That doesn't look like a valid image"));
      img.onload = () => {
        const scale = Math.min(1, maxSize / Math.max(img.width, img.height));
        const w = Math.round(img.width * scale);
        const h = Math.round(img.height * scale);
        const canvas = document.createElement("canvas");
        canvas.width = w;
        canvas.height = h;
        canvas.getContext("2d").drawImage(img, 0, 0, w, h);
        resolve(canvas.toDataURL("image/jpeg", 0.85));
      };
      img.src = reader.result;
    };
    reader.readAsDataURL(file);
  });
}

async function handleAvatarFile(file) {
  if (!file) return;
  if (!file.type.startsWith("image/")) {
    showToast("Please choose an image file", "error");
    return;
  }
  try {
    pendingAvatarDataUrl = await resizeImageFile(file);
    renderAvatarInto(document.getElementById("settings-avatar-preview"), {
      avatarDataUrl: pendingAvatarDataUrl,
    });
  } catch (err) {
    showToast(err.message, "error");
  }
}

async function saveProfile(e) {
  e.preventDefault();
  const name = document.getElementById("settings-name-input").value.trim();
  const email = document.getElementById("settings-email-input").value.trim();
  const btn = document.getElementById("save-profile-btn");

  if (!name || !email) {
    showToast("Name and email can't be empty", "error");
    return;
  }

  btn.disabled = true;
  btn.innerHTML = `<span class="spinner"></span> Saving`;
  try {
    currentProfile = await apiFetch("/api/me", {
      method: "PUT",
      body: { name, email, avatar_data_url: pendingAvatarDataUrl || "" },
    });
    Storage.save(Storage.getToken(), currentProfile.email);
    Storage.saveProfile(currentProfile.name, currentProfile.avatar_data_url);
    paintIdentity();
    showToast("Your profile is all set", "success");
  } catch (err) {
    showToast(err.message, "error");
  } finally {
    btn.disabled = false;
    btn.textContent = "Save changes";
  }
}

async function changePassword(e) {
  e.preventDefault();
  const current = document.getElementById("password-current").value;
  const next = document.getElementById("password-new").value;
  const confirm = document.getElementById("password-confirm").value;
  const btn = document.getElementById("change-password-btn");

  if (next !== confirm) {
    showToast("Those new passwords don't match", "error");
    return;
  }

  btn.disabled = true;
  btn.innerHTML = `<span class="spinner"></span> Updating`;
  try {
    await apiFetch("/api/me/password", {
      method: "PUT",
      body: { current_password: current, new_password: next },
    });
    showToast("Your password is updated", "success");
    document.getElementById("password-current").value = "";
    document.getElementById("password-new").value = "";
    document.getElementById("password-confirm").value = "";
  } catch (err) {
    showToast(err.message, "error");
  } finally {
    btn.disabled = false;
    btn.textContent = "Update password";
  }
}

function openDeleteAccountConfirm() {
  document.getElementById("delete-account-password").value = "";
  openSheet("delete-account-sheet");
}

function closeDeleteAccountConfirm() {
  closeSheet("delete-account-sheet");
}

async function confirmDeleteAccount(e) {
  e.preventDefault();
  const password = document.getElementById("delete-account-password").value;
  const btn = document.getElementById("confirm-delete-account-btn");

  btn.disabled = true;
  btn.innerHTML = `<span class="spinner"></span> Deleting`;
  try {
    await apiFetch("/api/me", { method: "DELETE", body: { password } });
    Storage.clear();
    window.location.href = "login.html?deleted=success";
  } catch (err) {
    showToast(err.message, "error");
    btn.disabled = false;
    btn.textContent = "Delete my account";
  }
}

/* ---------- Init ---------- */
function initDashboard() {
  requireAuth();
  initSearch();

  document.getElementById("fab-new-note").addEventListener("click", () => openEditor());
  document.getElementById("sidebar-new-note-btn").addEventListener("click", () => openEditor());
  document.getElementById("cancel-editor-btn").addEventListener("click", closeEditor);
  document.getElementById("note-form").addEventListener("submit", saveNote);

  document.getElementById("cancel-delete-btn").addEventListener("click", closeDeleteConfirm);
  document.getElementById("confirm-delete-btn").addEventListener("click", confirmDelete);

  document.getElementById("sheet-backdrop").addEventListener("click", closeAllSheets);

  // Account settings
  document.getElementById("account-row").addEventListener("click", openSettings);
  document.getElementById("close-settings-btn").addEventListener("click", closeSettings);
  document.getElementById("profile-form").addEventListener("submit", saveProfile);
  document.getElementById("password-form").addEventListener("submit", changePassword);
  document.getElementById("avatar-file-input").addEventListener("change", (e) => handleAvatarFile(e.target.files[0]));
  document.getElementById("change-photo-btn").addEventListener("click", () => document.getElementById("avatar-file-input").click());
  document.getElementById("settings-logout-btn").addEventListener("click", logout);

  document.getElementById("open-delete-account-btn").addEventListener("click", openDeleteAccountConfirm);
  document.getElementById("cancel-delete-account-btn").addEventListener("click", closeDeleteAccountConfirm);
  document.getElementById("delete-account-form").addEventListener("submit", confirmDeleteAccount);

  // Keep the editor panel's mobile/desktop presentation in sync on resize
  // (e.g. rotating a tablet, or resizing a browser window).
  window.addEventListener("resize", () => {
    if (editingNoteId === null && !isDesktop()) {
      closeSheet("editor-sheet");
    }
  });

  loadProfileIntoUI();
  loadNotes();

  if (isDesktop()) {
    document.getElementById("note-form").style.display = "none";
    document.getElementById("editor-empty").style.display = "flex";
  }
}
