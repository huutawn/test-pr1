# Production-ready Auth Demo (Go backend + Next.js frontend)

Project structure:
- backend/  (Go microservice with PostgreSQL)
- fe/       (Next.js frontend)

Features:
- JWT-based authentication with access and refresh tokens
- Registration, login, token refresh, and a protected /api/me endpoint
- Landing page + auth pages (login/register)

How to run:
- Prerequisites: Go 1.20+, PostgreSQL, Node.js (LTS), npm/yarn

1) Start backend
- cd backend
- go mod tidy
- go run .
- Backend always listens on port 8080
- Environment vars for local dev (examples):
  - JWT_SECRET=supersecret
  - DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
  - DB_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
  - DB_HOST=localhost
  - DB_PORT=5432
  - DB_NAME=app
  - DB_USERNAME=postgres
  - DB_PASSWORD=postgres

2) Start frontend
- cd fe
- npm install
- NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api npm run dev
- Frontend uses http://localhost:3000
- You can set the browser API base by exporting NEXT_PUBLIC_API_BASE_URL

Usage:
- Visit http://localhost:3000 to see the landing page.
- Use /login and /register to authenticate. The frontend stores the JWT in localStorage for simplicity; for production, consider HttpOnly cookies and a proper session mechanism.

LazyOps distributed-k3s notes:
- Browser-side calls should use the backend public path, not internal DNS. Set `NEXT_PUBLIC_API_BASE_URL=${API_BASE_URL}` and let it resolve to `/api`.
- Backend-to-database access should use managed placeholders. Prefer `DATABASE_URL=${DB_URL}` or `DB_URL=${DB_URL}`.
- If the backend later needs to call another internal service, use cluster-safe URLs such as `http://be:8080`, never `localhost`.
- Do not expose the managed PostgreSQL service publicly.
