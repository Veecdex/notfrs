# Paperz — Go + Postgres + Vanilla JS Notes App

A full-stack notes app:

- **Backend**: Go, `net/http` (no web framework), JWT auth, bcrypt password hashing, Postgres via `lib/pq`.
- **Frontend**: plain HTML/CSS/JS, no build step, no framework. Light/dark theme, mobile app–style UI, and a proper desktop layout (sidebar + docked editor) at wider widths.

## 1. What's in it

- Sign up / log in with JWT sessions
- Forgot password (see the security note below - it's simplified for a project without an email service)
- Account settings: edit name, email, profile photo, and password
- Delete account (with password confirmation)
- Notes: create, edit, delete, search
- Light/dark mode toggle, remembered across visits
- Toast notifications and loading spinners throughout, so nothing ever just "does nothing"

## 2. Project structure

```
notes-app/
├── backend/
│   ├── go.mod
│   ├── .env.example            # copy to .env and fill in
│   ├── cmd/server/main.go      # entrypoint - wires everything together
│   ├── internal/
│   │   ├── config/             # env var loading
│   │   ├── db/                 # Postgres connection
│   │   ├── models/             # User, Note structs
│   │   ├── auth/                # bcrypt hashing + JWT issue/verify
│   │   ├── middleware/          # JWT auth check, CORS
│   │   └── handlers/
│   │       ├── auth_handler.go     # register, login, forgot/reset password
│   │       ├── note_handler.go     # notes CRUD
│   │       └── profile_handler.go  # view/edit account, change password, delete account
│   └── migrations/              # raw SQL, run in order
└── frontend/
    ├── index.html               # the notes dashboard (sidebar + editor)
    ├── login.html
    ├── register.html
    ├── forgot-password.html
    ├── css/style.css
    └── js/
        ├── icons.js             # inline SVG icon set (incl. the Paperz logo mark)
        ├── theme.js             # dark mode toggle + persistence
        ├── toast.js             # toast notifications
        ├── api.js               # fetch wrapper + token/profile storage
        ├── auth.js              # login/register/forgot-password logic
        └── notes.js             # notes CRUD + account settings UI logic
```

## 3. Database setup

Run these three files, in order, in pgAdmin's Query Tool against your `notes_app` database:

1. `backend/migrations/001_create_users.sql`
2. `backend/migrations/002_create_notes.sql`
3. `backend/migrations/003_add_profile_fields.sql` — adds the `name` and `avatar_data_url` columns used by the profile features

If you already ran 001 and 002 from before, you only need to run 003 now.

## 4. Configure and run the backend

```bash
cd backend
cp .env.example .env
```

Fill in `.env` with your Postgres credentials and a random `JWT_SECRET` (`openssl rand -hex 32` works well).

```bash
go mod tidy
go run ./cmd/server
```

You should see `connected to database` then `listening on http://localhost:8080`.

### If something 404s or errors that "should" work

**The single most common cause: an old server process is still running.** Every code update needs the server fully stopped and restarted - `go run` does not hot-reload. If you've had a terminal open across several updates, kill it completely (Ctrl+C, and if it's stubborn, close that terminal) before running `go run ./cmd/server` again. A leftover process from an earlier version is indistinguishable from a "broken" one until you check this.

Every endpoint below was just verified end-to-end against this exact codebase (a real compiled build, a real Postgres database, real HTTP requests) - copy-paste these against your own running server if anything seems off, so you can tell in seconds whether it's the server or the browser/frontend side:

```bash
# Register (grab the token from the response for the next commands)
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com","password":"password123"}'

# Swap in your real token below
TOKEN="paste-your-token-here"

# View profile
curl http://localhost:8080/api/me -H "Authorization: Bearer $TOKEN"

# Change password (this is the one that was 404ing - it shouldn't now)
curl -X PUT http://localhost:8080/api/me/password \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d '{"current_password":"password123","new_password":"newpassword456"}'

# Edit profile
curl -X PUT http://localhost:8080/api/me \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"New Name","email":"test@example.com","avatar_data_url":""}'

# Notes
curl -X POST http://localhost:8080/api/notes \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Test note","content":"Hello"}'
curl http://localhost:8080/api/notes -H "Authorization: Bearer $TOKEN"

# Forgot password check
curl -X POST http://localhost:8080/api/forgot-password/check \
  -H "Content-Type: application/json" -d '{"email":"test@example.com"}'
```

If any of these 404 or error against a genuinely fresh `go run ./cmd/server`, that's a real bug - tell me exactly which command and what it returned, and I'll fix it immediately. If they all work here but the same action fails in the browser, the issue is almost certainly the frontend not pointing at the right `API_BASE_URL` in `frontend/js/api.js`, or a stale browser tab that still has old JS cached (hard refresh with Ctrl+Shift+R / Cmd+Shift+R).

## 5. Run the frontend

Same as before - it's static files, no build step:

- VS Code: right-click `frontend/index.html` → **Open with Live Server**, or
- `python3 -m http.server 5500` from inside `frontend/`

Make sure `FRONTEND_ORIGIN` in `backend/.env` matches whatever address that serves from (e.g. `http://127.0.0.1:5500`), then restart the Go server.

## 6. A note on the forgot-password flow

You asked for forgot-password to just check whether the email exists and then let the user set a new password right there, with no email link. That's what's implemented in `authHandler.CheckEmail` and `authHandler.ResetPassword`, and it's genuinely simpler to build since it needs no email-sending service.

**The tradeoff:** anyone who knows (or guesses) a user's email address can reset that user's password, because there's no proof they actually own that inbox. That's fine for you testing this solo, or a closed group of people who trust each other. It is **not** fine once real strangers can create accounts.

When you're ready to harden it, the standard fix is:
1. `CheckEmail` still checks existence, but the response never reveals the answer either way (always say "if that email exists, we've sent a link").
2. Generate a random token, store it with an expiry (a new `password_reset_tokens` table), and email a link containing that token using something like Postmark, Resend, or SendGrid.
3. `ResetPassword` verifies the token instead of just the email.

Happy to build that out when you're ready to wire up an email provider.

## 7. How the auth flow works (short version)

1. `POST /api/register` — hashes the password with bcrypt, stores the user (with name), returns a signed JWT.
2. `POST /api/login` — looks up the user by email, compares the bcrypt hash, returns a fresh JWT.
3. The frontend stores that JWT in `localStorage` and sends it as `Authorization: Bearer <token>` on every request to `/api/notes*` and `/api/me*`.
4. `middleware.RequireAuth` checks that header on every protected route, verifies the JWT, and puts the user's ID on the request context. Every notes/profile query is scoped to that user's ID.

Tokens expire after 24 hours (`tokenTTL` in `internal/auth/jwt.go`) - after that, log in again.

## 8. Profile photos: how they're stored

There's no object storage (S3, Cloudinary, etc.) wired up, so profile photos are resized client-side (down to ~240px) and stored as a base64 data URL directly in the `avatar_data_url` text column in Postgres. This keeps things simple and works fine for a personal project, but it's not how you'd do it at scale - a real app would upload the image to object storage and store just the URL. Worth revisiting if this ever needs to support lots of users or larger images.

---

## 9. Deploying to Railway (when you're ready)

You said this can wait, so here's the plan for when you want it - no need to do this now.

Railway is a good fit here because it can host both your Go API and a Postgres database, and it builds straight from a GitHub repo.

### Step 1: Push your code to GitHub

Railway deploys from a Git repo, so the project needs to be on GitHub first:

```bash
cd notes-app
git init
git add .
git commit -m "Initial commit"
```

Create a new (empty) repo on GitHub, then:

```bash
git remote add origin https://github.com/YOUR_USERNAME/YOUR_REPO.git
git branch -M main
git push -u origin main
```

**Before you push:** make sure `.env` is in a `.gitignore` file so your real secrets never end up on GitHub. Add this as `backend/.gitignore`:
```
.env
```

### Step 2: Create the Railway project

1. Go to [railway.app](https://railway.app) and sign up/log in (GitHub login is easiest).
2. Click **New Project** → **Deploy from GitHub repo** → pick your repo.
3. Railway will try to auto-detect a service. Since your Go code lives in `backend/`, not the repo root, you'll need to tell it that:
   - Open the new service's **Settings** tab.
   - Under **Root Directory**, set it to `backend`.
   - Under **Build**, Railway auto-detects Go via `go.mod` and runs `go build ./cmd/server` - if it doesn't, you can set a custom build command there.
   - Under **Deploy**, set the **Start Command** to `./server` (or whatever binary name your build produces - Railway usually figures this out automatically for Go projects).

### Step 3: Add a Postgres database

1. In the same Railway project, click **+ New** → **Database** → **Add PostgreSQL**.
2. Railway spins up a Postgres instance and gives it internal connection variables automatically (`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, plus a combined `DATABASE_URL`).

### Step 4: Wire the database into your app

Your Go code currently expects separate `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` variables (see `internal/config/config.go`). You have two options:

- **Easiest:** in your backend service's **Variables** tab on Railway, manually add `DB_HOST`, `DB_PORT`, etc., and reference Railway's Postgres variables using their `${{Postgres.PGHOST}}` style variable references (Railway shows you the exact names to reference under the Postgres service's **Variables** tab). Also set `DB_SSLMODE=require` (Railway's Postgres requires SSL), and set `JWT_SECRET` and `PORT` (Railway sets `PORT` automatically at runtime - you may not need to set it yourself, just make sure your code reads `os.Getenv("PORT")`, which it already does).
- **Cleaner long-term:** update `config.go` to also support reading a single `DATABASE_URL` connection string (Railway, Render, Heroku, and most hosts provide one), and prefer it when present. This is a small change - ask me when you're ready to deploy and I'll add it.

### Step 5: Run your migrations against the Railway database

Once the Postgres service is up, you need to run the three SQL files against it. Easiest way:
1. On the Postgres service in Railway, open the **Data** tab, which gives you a built-in query console - or copy the connection details and connect with pgAdmin like you do locally.
2. Run `001_create_users.sql`, `002_create_notes.sql`, `003_add_profile_fields.sql` in order.

### Step 6: Set `FRONTEND_ORIGIN` for CORS

Once your frontend has a real URL (see below), set `FRONTEND_ORIGIN` on the backend service to that exact URL, so the browser is allowed to call your API from there.

### Step 7: Deploy the frontend

Since the frontend is static HTML/CSS/JS with no build step, you have a few good, mostly-free options:
- **Railway itself** - add a second service pointing at the `frontend/` folder, or
- **Netlify** or **Vercel** - drag-and-drop the `frontend/` folder, or connect the same GitHub repo and point the "publish directory" at `frontend/`.

Whichever you pick, update `API_BASE_URL` in `frontend/js/api.js` from `http://localhost:8080` to your deployed backend's Railway URL.

### That's the shape of it

Once you're ready to actually do this, I can walk through it step by step with you and adjust `config.go` for `DATABASE_URL` support at the same time.
