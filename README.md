# GitHub Mirror

[![Go](https://img.shields.io/badge/Go_1.26-00ADD8?logo=go&logoColor=fff)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL_15-4169E1?logo=postgresql&logoColor=fff)](https://postgresql.org)
[![chi](https://img.shields.io/badge/chi_router-5e5e5e?logo=go&logoColor=fff)](https://github.com/go-chi/chi)
[![HTMX](https://img.shields.io/badge/HTMX_2-3366CC?logo=htmx&logoColor=fff)](https://htmx.org)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=fff)](https://docker.com)

**Download • [Quick Start](#run-locally) • [Docker](#docker) • [Guide](#usage-guide)**

---

Mirror GitHub repositories across organizations, accounts, or regions — with webhook-triggered real-time sync, optional cron scheduling, and a clean web UI.

> No more manual push/pull across orgs. Set it once, and branches + tags stay in sync automatically.

## Why GitHub Mirror

GitHub Mirror is built for teams that maintain repositories across multiple GitHub organizations or accounts and need to keep them in sync:

1. tell it where the source repo lives and where the target should be
2. push a branch to the source
3. the webhook fires, the worker runs, the target gets updated
4. go back to coding

No cron hack scripts, no CI pipelines pretending to be sync tools, no `git push --mirror` footguns.

## Features

| What | How |
|------|-----|
| **Branch & tag sync** | Mirror any branch or tag from source to target repo |
| **Webhook-triggered** | Push to source → webhook fires → target updates in seconds |
| **Cron scheduling** | Set `*/30 * * * *` and the scheduler handles the rest |
| **Branch filtering** | Only sync branches matching a pattern (`main`, `release/*`) |
| **Tag & delete sync** | Optionally sync tags and propagate branch deletions |
| **Web UI** | Dashboard, detail view, forms, schedule editor, setup guide |
| **JSON API** | Manage mirrors programmatically |
| **Sync history** | See every job result, retry attempts, and error messages |
| **Secure** | Tokens encrypted at rest with AES-GCM |

## Architecture

```mermaid
flowchart LR
    Browser["Browser<br/>(Web UI)"]
    Webhook["GitHub<br/>(Webhook)"]

    subgraph App ["GitHub Mirror"]
        Router["chi Router"]
        Worker["Worker<br/>(async sync)"]
        Scheduler["Scheduler<br/>(cron)"]
        GitOps["Git Ops<br/>(bare repo · fetch · push)"]
        Store[(Postgres)]
    end

    Target["GitHub<br/>(Target Repo)"]
    Source["GitHub<br/>(Source Repo)"]

    Browser --> Router
    Webhook --> Router
    Router --> Worker
    Router --> Scheduler
    Router --> Store
    Worker --> GitOps
    Scheduler --> GitOps
    GitOps --> Source
    GitOps --> Target
    Worker --> Store
    Scheduler --> Store
```

The app runs as three separate processes, all sharing the same Postgres database:

| Command | What it does |
|---------|-------------|
| `mirror server` | Serves the web UI and REST API. Also runs a built-in worker. |
| `mirror worker` | Standalone sync processor. Use this if the built-in one isn't enough. |
| `mirror scheduler` | Wakes up every minute, checks which mirrors are due for a cron sync, and enqueues jobs. |

## Quick Start

### Run locally

```bash
# 1. Prerequisites: Go 1.26+, PostgreSQL 15+, Git
git clone https://github.com/dat-lt-amira/github-mirror.git
cd github-mirror

# 2. Set up environment (or copy and tweak the values)
export DATABASE_URL="postgres://mirror:mirrorpass@localhost:5432/mirror?sslmode=disable"
export APP_ENCRYPTION_KEY="your-32-byte-encryption-key-here!!!!"
export BASIC_AUTH_USERNAME="admin"
export BASIC_AUTH_PASSWORD="admin123"

# 3. Start the server (web + API + background worker)
make run
```

Open http://localhost:8080 and log in.

### Docker

```bash
docker compose up -d
```

This starts three services: the web server, the sync worker, and the cron scheduler.

Open http://localhost:8080 — ready to go.

## Typical Workflow

**1. Create a mirror** — tell it the source repo, the target repo, and which branches to sync.

**2. Set up the webhook** — the UI shows you the webhook URL and secret. Paste them in your source repo's GitHub webhook settings.

**3. Push a branch** — `git push origin feature/xyz`. The webhook fires, the worker runs, and the target repo is updated within seconds.

**4. Check the history** — every sync job is logged. Green = success, red = something went wrong (with the error message to help debug).

**5. Add a schedule (optional)** — don't want webhooks? Set a cron expression like `0 * * * *` and let the scheduler run a full sync every hour.

## Usage Guide

### Create a Mirror

1. Open the dashboard and click **New Mirror**
2. Fill in the source repo and target repo (owner + repo name, or full GitHub URL)
3. Optionally add personal access tokens if the repos are private
4. Toggle sync options: tag syncing, delete propagation, force push
5. Optionally type a cron expression (like `*/10 * * * *`)
6. Click save — you'll land on the mirror detail page

### Set Up the Webhook

Every mirror gets a unique webhook URL. To set it up:

1. From the mirror detail page, click **Setup Guide**
2. Copy the webhook URL and the generated secret
3. Go to your source repo on GitHub → Settings → Webhooks → **Add webhook**
4. Paste the URL and secret, set content type to `application/json`
5. Select **Just the push event** (or include tag events if you need them)
6. Save it — next time someone pushes, the mirror updates automatically

### Scheduled Sync

No webhooks? No problem. Open the mirror's **Edit Schedule** page and type a cron expression:

| Expression | Meaning |
|------------|---------|
| `*/10 * * * *` | Every 10 minutes |
| `0 * * * *` | Every hour |
| `0 9 * * 1-5` | Every weekday at 9 AM UTC |
| *(blank)* | Manual only (no auto sync) |

The scheduler checks once per minute and won't enqueue a job if one is already running.

### Three Ways to Sync

- **Manual** — click the **Sync Now** button on the mirror detail page
- **Webhook** — push to the source repo, the webhook triggers an incremental sync
- **Scheduled** — cron expression fires a full branch sync automatically

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | Postgres connection string |
| `BASIC_AUTH_USERNAME` | ✅ | — | Web UI login username |
| `BASIC_AUTH_PASSWORD` | ✅ | — | Web UI login password |
| `APP_ENCRYPTION_KEY` | ✅ | — | AES-GCM key for encrypting tokens (must be 32 bytes) |
| `SERVER_ADDRESS` | ❌ | `:8080` | HTTP listen address |
| `MIRROR_CACHE_DIR` | ❌ | OS temp dir | Where bare repos are cached locally |
| `SCHEDULER_POLL_INTERVAL` | ❌ | `1m` | How often the scheduler wakes up |

## Tech Stack

- **Go 1.26** — chi router, sql, html/template
- **PostgreSQL 15** — primary store with embedded SQL migrations
- **HTMX** — dynamic UI without JavaScript frameworks
- **Docker** — multi-stage build (golang:1.26 → alpine with git)
- **AES-GCM** — credential encryption at rest
