# Workflow Builder Backend (Go)

Go HTTP API extracted from the Next.js `workflow-builder-main` app. Same PostgreSQL schema and route shapes so the frontend can switch with a base URL change.

## Stack

- Go 1.22+
- [chi](https://github.com/go-chi/chi) router
- [pgx](https://github.com/jackc/pgx) (PostgreSQL)
- [goja](https://github.com/dop251/goja) for `CODE_NODE` JS execution

## API surface (ported)

| Method | Path | Source |
|--------|------|--------|
| `GET` | `/api/workflows` | server action `onGetWorkflows` |
| `POST` | `/api/workflows` | server action `onCreateWorkflow` |
| `PUT` | `/api/workflows/{id}` | server action `onUpdateWorkflow` |
| `DELETE` | `/api/workflows/{id}` | server action `onDeleteWorkflow` |
| `GET` | `/api/workflows/{id}` | `app/api/workflows/[id]` |
| `POST` | `/api/workflows/{id}/addNode` | addNode route |
| `DELETE` | `/api/workflows/{id}/deleteNode` | deleteNode route |
| `PUT` | `/api/workflows/{id}/updateNodeAndEdges` | updateNodeAndEdges |
| `PUT` | `/api/workflows/{id}/updateNodeData` | updateNodeData |
| `GET` | `/api/workflows/{id}/getNodeData` | getNodeData |
| `GET` | `/api/workflows/{id}/executions` | executions list |
| `GET` | `/api/workflows/{id}/executions/{executionId}` | execution graph |
| `GET` | `/api/workflows/{id}/executions/{executionId}/getNodeData` | execution node data |
| `*` | `/api/webhooks/{id}` | webhook trigger + engine |
| `GET` | `/health` | health check |

## Run

```bash
cp .env.example .env
# set PG_DB_URL to your Postgres

# from this directory
go run ./cmd/server
```

On first start, if the `workflows` table is missing, `migrations/001_init.sql` is applied (same schema as Prisma / in-memory SQL).

```bash
# build binary
go build -o bin/server ./cmd/server
./bin/server
```

## Layout

```
cmd/server/          entrypoint
internal/config/     env config
internal/db/         pool + migrations
internal/models/     domain types
internal/handlers/   HTTP layer
internal/services/   business logic (workflows, nodes, executions, webhook engine, JS runner)
internal/middleware/ CORS + logging
migrations/          SQL schema
```

## Notes

- `DEFAULT_USER_ID` defaults to `"1"` to match the current Next.js hard-coded user.
- `CODE_NODE` runs in goja (pure Go). Scripts that `require('axios')` / `require('moment')` need pure-JS rewrites or a future Node runner.
- Frontend (`../workflow-builder-main`) uses `NEXT_PUBLIC_API_URL=http://localhost:8080` via `lib/api.ts`.
- CORS defaults to `http://localhost:3000`; set `CORS_ORIGINS` for other origins.
