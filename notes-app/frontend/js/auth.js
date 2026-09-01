function showInlineError(el, message) {
  el.innerHTML = `${Icons.alert}<span>${message}</span>`;
  el.classList.add("show");
}

function hideInlineError(el) {
  el.classList.remove("show");
}

function setButtonLoading(btn, loading, idleLabel) {
  if (loading) {
    btn.dataset.idleLabel = idleLabel;
    btn.disabled = true;
    btn.innerHTML = `<span class="spinner"></span> Please wait`;
  } else {
    btn.disabled = false;
    btn.textContent = idleLabel;
  }
}

function initLoginForm() {
  redirectIfLoggedIn();

  const form = document.getElementById("login-form");
  const errorBox = document.getElementById("error-box");
  const submitBtn = document.getElementById("submit-btn");

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    hideInlineError(errorBox);

    const email = document.getElementById("email").value.trim();
    const password = document.getElementById("password").value;

    setButtonLoading(submitBtn, true, "Log in");
    try {
      const data = await apiFetch("/api/login", {
        method: "POST",
        body: { email, password },
      });
      Storage.save(data.token, data.user.email);
      Storage.saveProfile(data.user.name, data.user.avatar_data_url);
      window.location.href = "index.html";
    } catch (err) {
      showInlineError(errorBox, err.message);
      setButtonLoading(submitBtn, false, "Log in");
    }
  });
}

function initRegisterForm() {
  redirectIfLoggedIn();

  const form = document.getElementById("register-form");
  const errorBox = document.getElementById("error-box");
  const submitBtn = document.getElementById("submit-btn");

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    hideInlineError(errorBox);

    const name = document.getElementById("name").value.trim();
    const email = document.getElementById("email").value.trim();
    const password = document.getElementById("password").value;
    const confirm = document.getElementById("confirm-password").value;

    if (password !== confirm) {
      showInlineError(errorBox, "Those passwords don't match - give it another shot.");
      return;
    }

    setButtonLoading(submitBtn, true, "Create account");
    try {
      const data = await apiFetch("/api/register", {
        method: "POST",
        body: { name, email, password },
      });
      Storage.save(data.token, data.user.email);
      Storage.saveProfile(data.user.name, data.user.avatar_data_url);
      window.location.href = "index.html";
    } catch (err) {
      showInlineError(errorBox, err.message);
      setButtonLoading(submitBtn, false, "Create account");
    }
  });
}

/* ---------- Forgot password (two-step: check email, then set new password) ---------- */
function initForgotPasswordForm() {
  redirectIfLoggedIn();

  const emailForm = document.getElementById("email-form");
  const resetForm = document.getElementById("reset-form");
  const errorBox = document.getElementById("error-box");
  const checkBtn = document.getElementById("check-btn");
  const resetBtn = document.getElementById("reset-btn");
  let confirmedEmail = "";

  emailForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    hideInlineError(errorBox);
    const email = document.getElementById("email").value.trim();

    checkBtn.disabled = true;
    checkBtn.innerHTML = `<span class="spinner"></span> Checking`;
    try {
      const data = await apiFetch("/api/forgot-password/check", {
        method: "POST",
        body: { email },
      });
      if (!data.exists) {
        showInlineError(errorBox, "We couldn't find an account with that email. Double-check it, or create a new account instead.");
        return;
      }
      confirmedEmail = email;
      document.getElementById("step-email").style.display = "none";
      document.getElementById("step-reset").style.display = "block";
      document.getElementById("found-email-label").textContent = email;
    } catch (err) {
      showInlineError(errorBox, err.message);
    } finally {
      checkBtn.disabled = false;
      checkBtn.textContent = "Continue";
    }
  });

  resetForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    hideInlineError(errorBox);

    const newPassword = document.getElementById("new-password").value;
    const confirmPassword = document.getElementById("confirm-new-password").value;

    if (newPassword !== confirmPassword) {
      showInlineError(errorBox, "Those passwords don't match - give it another shot.");
      return;
    }

    resetBtn.disabled = true;
    resetBtn.innerHTML = `<span class="spinner"></span> Updating`;
    try {
      await apiFetch("/api/forgot-password/reset", {
        method: "POST",
        body: { email: confirmedEmail, new_password: newPassword },
      });
      window.location.href = "login.html?reset=success";
    } catch (err) {
      showInlineError(errorBox, err.message);
    } finally {
      resetBtn.disabled = false;
      resetBtn.textContent = "Update password";
    }
  });
}
