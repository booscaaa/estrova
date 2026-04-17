# Estrova for Claude Code

A Model Context Protocol (MCP) server that integrates your Strava fitness data with Claude Code. Get AI-powered training plans, performance analysis, and a web dashboard — all driven by your real Strava history.

## What it does

- Authenticates with Strava via OAuth2 and syncs your activities to a local SQLite database
- Exposes 16 MCP tools so Claude can read your profile, stats, HR zones, activities, and training goals
- Generates and stores personalized multi-week training plans based on your history
- Detects scheduling conflicts across multiple concurrent goals
- Serves a web dashboard at `http://localhost:3030` for visual plan management

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Getting Strava API Credentials](#getting-strava-api-credentials)
3. [Installation](#installation)
   - [Option A — Install via GitHub (recommended)](#option-a--install-via-github-recommended)
   - [Option B — Build from source](#option-b--build-from-source)
4. [Configuring Claude Code](#configuring-claude-code)
5. [First-time Authentication](#first-time-authentication)
6. [Syncing Activities](#syncing-activities)
7. [Creating a Training Goal and Plan](#creating-a-training-goal-and-plan)
8. [Web Dashboard](#web-dashboard)
9. [Available MCP Tools](#available-mcp-tools)
10. [Database](#database)

---

## Prerequisites

- [Claude Code](https://claude.ai/code) installed
- A [Strava](https://www.strava.com) account

No other runtime is required — pre-built binaries are provided for Linux, macOS, and Windows.

---

## Getting Strava API Credentials

You need to register an application on Strava to get the credentials used during authentication. This is a one-time setup.

### Step 1 — Access the Strava API settings

Go to [https://www.strava.com/settings/api](https://www.strava.com/settings/api).

If prompted, log in to your Strava account first.

### Step 2 — Create your application

Fill in the form with the following values:

| Field | Value |
|-------|-------|
| **Application Name** | Anything you like, e.g. `My Claude Coach` |
| **Category** | Choose any (e.g. `Other`) |
| **Club** | Leave blank |
| **Website** | `http://localhost` |
| **Application Description** | Optional |
| **Authorization Callback Domain** | `localhost` |

Click **Create** (or **Update** if a previous app already exists).

### Step 3 — Copy your credentials

After saving, the page shows your application details. Note down:

- **Client ID** — a short number, e.g. `152485`
- **Client Secret** — a long hex string

> These credentials are sensitive. Never share them or commit them to a repository.

### Step 4 — Upload an app icon (optional)

Strava requires an icon before the OAuth consent screen works. Upload any image (PNG/JPG, minimum 124×124 px) in the **App Icon** field and save.

---

> **Note:** The authorization callback URL used by this MCP server is `http://localhost:8765/callback`. Strava only checks the **domain** (`localhost`), so no further URL configuration is needed.

---

## Installation

Pre-built binaries are published automatically on every release. No Go installation required.

### Linux / macOS (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/booscaaa/estrova/main/install.sh | bash
```

This script detects your OS and architecture, downloads the correct binary from the latest GitHub release, and places it at `/usr/local/bin/estrova`.

### Manual download

1. Go to the [Releases page](https://github.com/booscaaa/estrova/releases/latest)
2. Download the archive for your platform:

| Platform | File |
|----------|------|
| Linux x86-64 | `estrova_linux_amd64.tar.gz` |
| Linux ARM64 | `estrova_linux_arm64.tar.gz` |
| macOS x86-64 (Intel) | `estrova_darwin_amd64.tar.gz` |
| macOS ARM64 (Apple Silicon) | `estrova_darwin_arm64.tar.gz` |
| Windows x86-64 | `estrova_windows_amd64.zip` |

3. Extract and move the binary:

```bash
# Linux / macOS
tar -xzf estrova_linux_amd64.tar.gz
sudo mv estrova /usr/local/bin/
chmod +x /usr/local/bin/estrova
```

```powershell
# Windows — extract the zip, then move estrova.exe to a folder on your PATH
# e.g. C:\Users\<you>\bin\estrova.exe
```

4. Verify:

```bash
estrova
# Estrova MCP server iniciado — Web UI: http://localhost:3030
```

### Build from source (optional)

Only needed if you want to modify the code:

```bash
git clone https://github.com/booscaaa/estrova.git
cd estrova
go build -o estrova .
sudo mv estrova /usr/local/bin/
```

---

## Configuring Claude Code

### Global configuration (all projects)

Edit `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "estrova": {
      "command": "estrova",
      "env": {
        "STRAVA_CLIENT_ID": "your_client_id",
        "STRAVA_CLIENT_SECRET": "your_client_secret"
      }
    }
  }
}
```

Replace `estrova` with the full path to the binary if it is not on your `$PATH`.

### Project-level configuration

Create or edit `.mcp.json` in the root of your project:

```json
{
  "mcpServers": {
    "estrova": {
      "command": "/usr/local/bin/estrova",
      "env": {
        "STRAVA_CLIENT_ID": "your_client_id",
        "STRAVA_CLIENT_SECRET": "your_client_secret"
      }
    }
  }
}
```

> **Tip:** Add `.mcp.json` to `.gitignore` so your credentials are never committed.

### Verifying the MCP server is connected

Start Claude Code and run:

```
/mcp
```

You should see `estrova` listed as a connected server. If not, check the path to the binary and that your environment variables are set.

---

## First-time Authentication

The very first thing you need to do is authenticate with Strava. In Claude Code, just ask:

```
authenticate with strava
```

Claude will call `estrova_authenticate`, which:

1. Starts a local callback server on `http://localhost:8765/callback`
2. Opens your browser to the Strava OAuth consent page
3. After you approve, exchanges the code for an access token
4. Saves the token to `~/.estrova.db` (SQLite)

The token is automatically refreshed whenever it expires — you only need to authenticate once.

**Check authentication status:**

```
what's my strava auth status?
```

---

## Syncing Activities

After authenticating, sync your Strava activities to the local database:

```
sync my strava activities
```

By default, this fetches up to 5 pages (1 000 activities). To fetch more:

```
sync my strava activities, fetch 10 pages
```

Synced activities are stored locally and matched automatically against your training plan sessions.

---

## Creating a Training Goal and Plan

### 1. Create a goal

```
create a strava goal: Marathon in October 2026
```

Claude will call `estrova_create_goal` with parameters like:

| Parameter | Example |
|-----------|---------|
| `name` | `Marathon 2026` |
| `sport_type` | `Run` |
| `target_type` | `distance` |
| `target_value` | `42.2` |
| `target_date` | `2026-10-15` |

### 2. Generate a training plan

```
generate a training plan for my Marathon 2026 goal
```

Claude will:

1. Call `estrova_analyze_for_goal` — fetches your recent activities, HR zones, other goal sessions, and scheduling constraints
2. Use that data to build a personalized multi-week plan
3. Call `estrova_save_plan` — persists the plan to the database, linked to your goal

### 3. View your plan

```
show me my training plan for Marathon 2026
```

Or open the [web dashboard](#web-dashboard) at `http://localhost:3030`.

### 4. Resolve conflicts

If you have multiple concurrent goals, sessions from different plans may clash on the same day:

```
are there any conflicts in my training plans?
```

Claude will call `estrova_list_conflicts` and help you reschedule sessions.

---

## Web Dashboard

The web server starts automatically when the MCP server launches.

Open [http://localhost:3030](http://localhost:3030) in your browser.

**Features:**

- Goals overview with progress (completed / total sessions)
- Weekly training plan view with session details
- Activities list with sync status
- Conflict detector across all active goals
- Edit individual sessions (type, pace, HR zone, distance, duration)
- Dashboard with weekly volume and pace trend charts

---

## Available MCP Tools

### Authentication

| Tool | Description |
|------|-------------|
| `estrova_authenticate` | OAuth2 login — opens browser, saves token |
| `estrova_auth_status` | Check token validity and synced activity count |

### Activities

| Tool | Parameters | Description |
|------|------------|-------------|
| `estrova_sync` | `pages` (default 5) | Fetch & sync activities from Strava |
| `estrova_list_activities` | `type`, `after`, `before`, `limit` | Query local database |
| `estrova_get_activity` | `activity_id` | Full activity detail (laps, segments, best efforts) |

### Athlete Profile

| Tool | Description |
|------|-------------|
| `estrova_get_athlete` | Name, city, country, premium status |
| `estrova_get_athlete_stats` | Run/bike/swim totals (recent, YTD, all-time) |
| `estrova_get_athlete_zones` | Heart rate and power zones |

### Goals & Plans

| Tool | Parameters | Description |
|------|------------|-------------|
| `estrova_create_goal` | `name`, `sport_type`, `target_type`, `target_value`, `target_date` | Create a new training goal |
| `estrova_list_goals` | — | List all goals with progress |
| `estrova_delete_goal` | `goal_id` | Remove goal and its plan |
| `estrova_analyze_for_goal` | `goal_id` | Collect context for plan generation |
| `estrova_save_plan` | `goal_id`, `plan_json` | Persist generated plan to database |
| `estrova_get_plan` | `goal_id` | Retrieve plan organized by week |
| `estrova_list_conflicts` | — | Detect scheduling conflicts across goals |
| `estrova_update_session` | `session_id`, fields | Edit a session in the plan |

---

## Database

All data is stored in a single SQLite file at:

```
~/.estrova.db
```

It is created automatically on first run. Tables:

| Table | Contents |
|-------|----------|
| `tokens` | OAuth2 access/refresh tokens |
| `athlete` | Cached athlete profile |
| `activities` | Synced Strava activities |
| `goals` | Training goals |
| `plan_sessions` | Individual sessions per goal (weeks / workouts) |

To inspect it directly:

```bash
sqlite3 ~/.estrova.db ".tables"
sqlite3 ~/.estrova.db "SELECT name, target_date FROM goals;"
```

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `STRAVA_CLIENT_ID` | Yes | Client ID from Strava API settings |
| `STRAVA_CLIENT_SECRET` | Yes | Client secret from Strava API settings |

---

## License

MIT
