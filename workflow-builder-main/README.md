# Workflow Builder (Next.js frontend)

UI for the workflow builder. **API traffic goes to the Go backend** (`../workflow-builder-backend`).

## Setup

```bash
cp .env.example .env
# NEXT_PUBLIC_API_URL=http://localhost:8080  (default)

npm install
```

Start the Go API first:

```bash
cd ../workflow-builder-backend
cp .env.example .env   # set PG_DB_URL
go run ./cmd/server
```

Then the frontend:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Architecture

| Layer | Location |
|-------|----------|
| Frontend (Next.js 14) | this package |
| Backend API (Go) | `../workflow-builder-backend` |
| Database | PostgreSQL (`PG_DB_URL` on the Go service) |

Client and server components call the Go API through `lib/api.ts` (`apiFetch` / `apiUrl`).  
Legacy routes under `app/api/*` and Prisma are unused by the UI; keep them only as reference until removed.

## Env

| Variable | Purpose |
|----------|---------|
| `NEXT_PUBLIC_API_URL` | Base URL of the Go backend (browser + server) |
| `API_URL` | Optional server-only override |

Webhook URLs shown in the editor use the same base (`/api/webhooks/{path}` on the Go server).
