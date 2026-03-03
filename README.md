# HR Management System (HR Boshqaruv Tizimi)

## Overview
HR Management System with Hikvision attendance tracking integration. Go backend + React/TypeScript frontend + PostgreSQL database.

## Architecture
- **Backend**: Go (net/http) server in `main.go` + `goserver/` package
- **Frontend**: React + TypeScript + Vite + TailwindCSS + shadcn/ui in `client/`
- **Database**: PostgreSQL (auto-creates tables on startup)
- **Auth**: Cookie-based session auth (NOT Replit Auth)

## Key Files
- `main.go` - Entry point, starts Go HTTP server on port 5000
- `goserver/routes.go` - All API route handlers
- `goserver/database.go` - PostgreSQL connection and table creation
- `goserver/storage.go` - Data access layer
- `goserver/session.go` - Session/auth management
- `client/src/App.tsx` - React app entry with routing
- `vite.config.ts` - Vite build configuration
- `package.json` - Node.js dependencies for frontend

## Running
1. Build frontend: `npx vite build` (outputs to `dist/public/`)
2. Build Go binary: `go build -o hr-system main.go`
3. Run: `PORT=5000 ./hr-system`

The workflow command is: `cd /home/runner/workspace && PORT=5000 ./hr-system`

## Default Credentials
- sudo / sudo123 (superadmin)
- admin / admin123 (admin)

## Features
- Employee management (CRUD)
- Group management
- Attendance tracking via Hikvision cameras
- Telegram bot notifications
- Role-based access (sudo/admin roles)
- Dark mode UI

## Dependencies
- Go 1.25: github.com/lib/pq, golang.org/x/crypto
- Node.js: React, Vite, TailwindCSS, shadcn/ui, TanStack Query, wouter
