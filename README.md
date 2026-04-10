# Production-ready Auth Demo (Go backend + Next.js frontend)

Project structure:
- backend/  (Go, Gin-less microservice with SQLite)
- fe/       (Next.js frontend)

Features:
- JWT-based authentication with access and refresh tokens
- Registration, login, token refresh, and a protected /me endpoint
- Landing page + auth pages (login/register)

How to run:
- Prerequisites: Go 1.20+, Node.js (LTS), npm/yarn

1) Start backend
- cd backend
- go mod tidy
- go run .
- Backend listens on port from PORT env (default 8080)
- Uses SQLite DB at DB_PATH (default db.sqlite) in project root
- Environment vars (examples):
  - JWT_SECRET=supersecret
  - DB_PATH=db.sqlite

2) Start frontend
- cd fe
- npm install
- NEXT_PUBLIC_BACKEND_URL=http://localhost:8080 npm run dev
- Frontend uses http://localhost:3000
- You can set the backend URL by exporting NEXT_PUBLIC_BACKEND_URL

Usage:
- Visit http://localhost:3000 to see the landing page.
- Use /login and /register to authenticate. The frontend stores the JWT in localStorage for simplicity; for production, consider HttpOnly cookies and a proper session mechanism.
