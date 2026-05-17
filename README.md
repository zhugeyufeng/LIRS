# LIRS 2026

LIRS is a runnable lab instrument reservation MVP.

## Stack

- Frontend: TypeScript, Next.js App Router, React, Tailwind CSS, shadcn-style UI primitives, Lucide icons
- Backend: Go + Gin core API, Hono API gateway, Zod validation, Drizzle ORM for typed PostgreSQL access
- Database: PostgreSQL 15
- Cache / queue base: Redis 7

## Run

Copy `.env.example` to `.env` first, then adjust passwords if needed.

```bash
make dev
```

Open `http://localhost:3000`. The Hono API gateway is exposed at `http://localhost:8090`; the Go/Gin core API is exposed at `http://localhost:8081`.

The Go service runs schema migration and seeds demo data on startup.
Docker Compose also starts a PostgreSQL backup sidecar that writes daily dumps into the `postgres-backups` volume and keeps the last 14 days.

## Initial Admin

Initial super admin credentials are intentionally not shown on the login page. Read the active values from the deployment environment:

- Email: `admin@lirs.local`
- Password: value of `INITIAL_ADMIN_PASSWORD` in `.env` or the Go API service environment

Override it with `INITIAL_ADMIN_EMAIL`, `INITIAL_ADMIN_PASSWORD`, and `INITIAL_ADMIN_NAME` on the Go API service.

## Test User

The seed creates an active ordinary user for frontend role testing:

- Email: `testuser@lirs.local`
- Password: value of `INITIAL_DEMO_USER_PASSWORD` in `.env` or the Go API service environment

## Local Checks

```bash
make test
```

`lirs_v37/` remains in the repository as the visual reference used during the migration.
