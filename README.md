# Lab Scheduling

Lab Scheduling coordinates shared training-center equipment, rooms, consumables and safety windows. It provides authenticated teacher, student, administrator and safety-owner workflows for requests, approvals, reservations, check-out/return, maintenance blocks, incidents and auditable recovery.

## Run

```bash
GOTOOLCHAIN=local go test ./... -count=1
GOTOOLCHAIN=local go run .
```

The service uses SQLite as a real relational database and applies versioned SQL migrations on startup. Set `DB_PATH`, `HTTP_ADDR`, `MIGRATIONS_PATH` and `SESSION_TTL` to configure it. `POST /api/v1/auth/login` accepts `{ "email": "admin@example.com", "password": "admin" }` for seeded users. Health endpoints are `/healthz` and `/readyz`.

## Domain flows

Teachers create course batches and booking requests. Administrators approve requests while capacity, qualification and maintenance windows are checked in one transaction. Students can request personal make-up sessions. Check-out and return consume and restore consumables. Safety owners can record incidents that suspend affected reservations and trigger compensation sessions. Every mutation is idempotent when an `Idempotency-Key` is supplied and creates an audit event.
