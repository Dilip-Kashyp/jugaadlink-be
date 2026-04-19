# JugaadLink Backend

**High-performance URL shortener API built with Go, Gin, PostgreSQL, and Redis.**

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Gin](https://img.shields.io/badge/Gin-Framework-00ADD8)](https://gin-gonic.com)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)](https://redis.io)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://docker.com)

JugaadLink is a production-ready URL shortener with JWT authentication, click analytics, geographic tracking, password-protected links, and Redis-backed sub-millisecond redirects. It is designed to be self-hosted and extended.

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Redirect Flow](#redirect-flow)
- [Contributing](#contributing)
- [License](#license)

## Features

| Feature | Description |
| :--- | :--- |
| Authentication | JWT-based login and registration with Bcrypt password hashing |
| URL Shortening | Generate unique 8-character short codes with automatic OG metadata fetching |
| Custom Slugs | Define your own short alias (3 to 20 characters, uniqueness enforced) |
| Password Protection | Bcrypt-protected links with a dedicated frontend verification flow |
| Link Expiry | Time-based (`expires_at`) and click-based (`max_clicks`) expiration |
| Link Toggle | Activate or deactivate any link; deactivated links redirect to a disabled page |
| Link Editing | Update password, expiry, click limit, tags, category, and comment after creation |
| Analytics | Per-link and dashboard-wide stats: clicks, devices, OS, browsers, countries, referrers |
| Geographic Tracking | IP-based country and city detection using ip-api.com |
| Tags and Metadata | Tags, categories, comments, and custom domains per link |
| Link Previews | OpenGraph metadata extraction for rich link cards |
| Redis Caching | Sub-millisecond redirect resolution for frequently accessed links |
| Guest Sessions | Anonymous link creation via session tokens, no account required |
| Seed Data | Generate 200 realistic URLs with click history for local testing |

## Project Structure

```
Url shortener-be/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── config/              # Database, Redis, and environment setup
│   ├── handler/             # HTTP route registration
│   ├── middleware/          # Auth, rate limiting, CORS, identity resolution
│   ├── models/              # GORM models: URL, Click, User, GuestSession
│   ├── service/             # Business logic: shorten, redirect, analytics, update
│   └── util/                # Response wrappers, helpers
├── docker-compose.yml       # PostgreSQL + Redis + API service definitions
├── Dockerfile               # Multi-stage Go build
├── Makefile                 # Shortcuts for dev, prod, and test commands
└── .env                     # Environment configuration
```

## Getting Started

### With Docker (Recommended)

Requires Docker and Docker Compose installed.

```bash
git clone https://github.com/your-username/jugaadlink-be.git
cd jugaadlink-be

cp .env.example .env
# Edit .env and set DB_PASSWORD, JWT_SECRET, and SERVER_URL

docker compose up --build
```

The API will be available at `http://localhost:8080`.

### Local Development

Requires Go 1.22+ and running PostgreSQL and Redis instances.

```bash
# Download Go module dependencies
go mod download

# Run the server with hot-reload (requires Air: https://github.com/air-verse/air)
make dev

# Or run directly without hot-reload
go run cmd/main.go
```

### Makefile Commands

| Command | Description |
| :--- | :--- |
| `make dev` | Start with Air hot-reload |
| `make up` | Start via Docker Compose |
| `make down` | Stop Docker Compose services |
| `make seed` | Populate the database with test data |

## Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL username | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | required |
| `DB_NAME` | Database name | `url_shortener` |
| `REDIS_URL` | Redis connection string | `localhost:6379` |
| `JWT_SECRET` | Secret key for signing JWTs | required |
| `SERVER_URL` | Base URL prepended to short codes (include trailing slash) | `http://localhost:8080/` |
| `FRONTEND_URL` | Frontend URL used for redirect fallback pages | `http://localhost:3000` |
| `PORT` | Port the HTTP server listens on | `8080` |

## API Reference

### Public Routes

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/:code` | Redirect to the original URL |
| `POST` | `/verify-password/:code` | Verify password for a protected link |
| `GET` | `/api/v1/test/ping` | Health check |

### URL Management `/api/v1/url`

| Method | Path | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/shorten` | Session or JWT | Create a short URL |
| `GET` | `/history` | Session or JWT | List all links for the current user or session |
| `DELETE` | `/:code` | JWT | Permanently delete a link |
| `PATCH` | `/:code/toggle` | Session or JWT | Activate or deactivate a link |
| `PATCH` | `/:code` | Session or JWT | Update link properties (password, expiry, tags, etc.) |
| `GET` | `/preview?url=...` | Session or JWT | Fetch OpenGraph metadata for a URL |

### Analytics `/api/v1/url`

| Method | Path | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/analytics` | JWT | Aggregate dashboard analytics |
| `GET` | `/analytics/:code` | JWT | Per-link click analytics |

### Auth `/api/v1/user`

| Method | Path | Description |
| :--- | :--- | :--- |
| `POST` | `/register` | Create a new account |
| `POST` | `/login` | Authenticate and receive a JWT |
| `GET` | `/get-user` | Get the currently authenticated user |

### Request Example

```bash
# Shorten a URL with a custom alias and password
curl -X POST http://localhost:8080/api/v1/url/shorten \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://github.com/your-username/jugaadlink-be",
    "custom_slug": "my-repo",
    "password": "secret123",
    "max_clicks": 100,
    "category": "GitHub"
  }'
```

## Redirect Flow

When a user visits a short link, the server evaluates the following conditions in order before issuing a redirect:

```
User visits /<code>
        |
        v
  Is link active?
   No  --> redirect to /link-disabled?code=<code>
   Yes --> continue
        |
        v
  Is link expired (time)?
   Yes --> redirect to /link-disabled?code=<code>&reason=expired
   No  --> continue
        |
        v
  Has click limit been reached?
   Yes --> redirect to /link-disabled?code=<code>&reason=limit
   No  --> continue
        |
        v
  Is link password-protected?
   Yes --> redirect to /password/<code>
   No  --> continue
        |
        v
  302 redirect to originalURL
  + record click asynchronously
  + cache in Redis (if no password, limit, or expiry)
```

## Contributing

Contributions in any form are welcome: bug reports, feature requests, documentation improvements, or pull requests. Please open an issue first for significant changes so the approach can be discussed before implementation.

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/your-feature`.
3. Commit your changes with a clear message.
4. Push to the branch and open a pull request against `main`.
