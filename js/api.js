// Talks to the Go backend. Change this if your API runs somewhere else.
const API_BASE_URL = "https://noters-fkae.onrender.com";

const Storage = {
  getToken: () => localStorage.getItem("notes_token"),
  getEmail: () => localStorage.getItem("notes_email"),
  getName: () => localStorage.getItem("notes_name") || "",
  getAvatar: () => localStorage.getItem("notes_avatar") || "",
  save: (token, email) => {
    localStorage.setItem("notes_token", token);
    localStorage.setItem("notes_email", email);
  },
  saveProfile: (name, avatarDataUrl) => {
    localStorage.setItem("notes_name", name || "");
    localStorage.setItem("notes_avatar", avatarDataUrl || "");
  },
  clear: () => {
    localStorage.removeItem("notes_token");
    localStorage.removeItem("notes_email");
    localStorage.removeItem("notes_name");
    localStorage.removeItem("notes_avatar");
  },
};

/** True if an API error message indicates the session needs a fresh login. */
function isSessionError(message) {
  const m = message.toLowerCase();
  return m.includes("token") || m.includes("session") || m.includes("authorization");
}

/**
 * apiFetch wraps fetch() with the API base URL, JSON headers, and the
 * bearer token (when present). It throws an Error with a readable
 * message on any non-2xx response, and on responses that don't parse
 * as JSON at all (e.g. hitting the wrong server on a busy port).
 */
async function apiFetch(path, { method = "GET", body } = {}) {
  const headers = { "Content-Type": "application/json" };
  const token = Storage.getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  let response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (networkErr) {
    throw new Error("Can't reach the server. Is the Go API running?");
  }

  if (response.status === 204) return null;

  let data = null;
  let parseFailed = false;
  try {
    data = await response.json();
  } catch (_) {
    parseFailed = true;
  }

  if (!response.ok) {
    const message = (data && data.error) || `Request failed (${response.status})`;
    throw new Error(message);
  }

  if (parseFailed) {
    throw new Error("Server sent back something unexpected. Check the API logs.");
  }

  return data;
}

/**
 * requireAuth redirects to the login page if there's no token stored.
 * Call this at the top of any page that needs a logged-in user.
 */
function requireAuth() {
  if (!Storage.getToken()) {
    window.location.href = "login.html";
  }
}

/** redirectIfLoggedIn sends already-authenticated users to the dashboard. */
function redirectIfLoggedIn() {
  if (Storage.getToken()) {
    window.location.href = "index.html";
  }
}

function logout() {
  Storage.clear();
  window.location.href = "login.html";
}
