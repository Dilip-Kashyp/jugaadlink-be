# 🔗 JugaadLink Backend — URL Shortener API

> High-performance URL shortener API built with Go, Gin, GORM, PostgreSQL, and Redis. Powers the JugaadLink platform with authentication, analytics, link protection, and geographic tracking.

![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)
![Gin](https://img.shields.io/badge/Gin-Framework-00ADD8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| **🔐 Authentication** | JWT-based auth with Bcrypt password hashing |
| **⚡ URL Shortening** | Generate 8-char unique short codes with metadata fetching |
| **🔒 Password Protection** | Bcrypt-protected links with dedicated frontend verify flow |
| **⏱️ Link Expiry** | Time-based (`expires_at`) and click-based (`max_clicks`) expiration |
| **⚙️ Link Toggle** | Activate/deactivate links — deactivated links redirect to frontend disabled page |
| **📊 Analytics** | Clicks, devices, OS, browsers, countries, referrers, timeline trends |
| **🌍 Geographic Tracking** | IP-based country detection using GeoLite2 |
| **🏷️ Tags & Metadata** | Tags, categories, comments, and custom domains per link |
| **🔗 Link Previews** | OpenGraph metadata extraction for rich link cards |
| **💨 Redis Caching** | Sub-millisecond redirect resolution for popular links |
| **🌱 Seeding** | Generate 200 dummy URLs with realistic click data for testing |

---

## 📡 API Endpoints

### Public Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/:code` | Redirect to original URL |
| `POST` | `/verify-password/:code` | Verify password for protected links |
| `GET` | `/api/v1/ping` | Health check |

### URL Management (`/api/v1/url`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/shorten` | Session/JWT | Create a short URL |
| `GET` | `/history` | Session/JWT | List all user's URLs |
| `DELETE` | `/:code` | JWT | Delete a URL |
| `PATCH` | `/:code/toggle` | Session/JWT | Activate/deactivate a URL |
| `GET` | `/preview?url=...` | Session/JWT | Fetch link metadata |

### Analytics (`/api/v1/url`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/analytics` | JWT | Dashboard-wide analytics |
| `GET` | `/analytics/:code` | JWT | Per-link analytics |

### Auth (`/api/v1/user`)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/register` | Create account |
| `POST` | `/login` | Login + JWT |
| `GET` | `/get-user` | Get current user |

---

## 🏗️ Project Structure

```
Url shortener-be/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── config/              # DB, Redis, environment setup
│   ├── handler/             # Route definitions
│   ├── middleware/           # Auth, Rate limiting, CORS
│   ├── models/              # GORM models (URL, Click, User)
│   ├── service/             # Business logic (shorten, redirect, analytics)
│   └── util/                # Helpers, response wrappers
├── docker-compose.yml       # PostgreSQL + Redis + API
├── Dockerfile               # Multi-stage Go build
├── Makefile                 # dev / prod / test shortcuts
└── .env                     # Environment config
```

---

## 🚀 Getting Started

### With Docker (Recommended)

```bash
# Clone
git clone https://github.com/your-username/jugaadlink-be.git
cd jugaadlink-be

# Copy env
cp .env.example .env

# Start everything
docker compose up --build
```

API runs on `http://localhost:8080`.

### Local Development

```bash
# Prerequisites: Go 1.22+, PostgreSQL, Redis

# Install deps
go mod download

# Run with hot-reload (requires Air)
make dev

# Or standard run
go run cmd/main.go
```

---

## ⚙️ Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | — |
| `DB_NAME` | Database name | `url_shortener` |
| `REDIS_URL` | Redis connection URL | `localhost:6379` |
| `JWT_SECRET` | JWT signing key | — |
| `SERVER_URL` | Base URL for short links | `http://localhost:8080/` |
| `PORT` | Server port | `8080` |

---

## 🔄 Redirect Flow

```
User clicks short link
       │
       ▼
 ┌─ Is Active? ──── No ──→ Redirect to /link-disabled
 │
 ├─ Expired? ─────── Yes ─→ Redirect to /link-disabled?reason=expired
 │
 ├─ Click Limit? ─── Yes ─→ Redirect to /link-disabled?reason=limit
 │
 ├─ Password? ────── Yes ─→ Redirect to /password/:code
 │
 └─ ✅ Valid ──────────────→ 302 Redirect to original URL
                              + Record click async
                              + Cache in Redis
```

---

## 📝 License

MIT © JugaadLink Team