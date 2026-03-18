# Apartment Management API - Backend

A comprehensive apartment/building management REST API built with Go, featuring real-time WebSocket communication, JWT authentication, role-based access control, and PostgreSQL storage.

## Tech Stack

| Technology | Version | Purpose |
|---|---|---|
| **Go** | 1.25 | Backend language |
| **Fiber** | v2.52.5 | HTTP framework |
| **PostgreSQL** | 16 | Primary database |
| **Redis** | 7 | Caching, rate limiting, sessions |
| **MinIO** | Latest | S3-compatible file storage |
| **JWT** | v5 | Authentication tokens |
| **WebSocket** | Fiber WS | Real-time events |
| **Docker** | Compose v2 | Containerization |
| **golang-migrate** | v4.17 | Database migrations |
| **Zap** | v1.27 | Structured logging |
| **Viper** | v1.18 | Configuration management |

## Architecture

```
apartment-backend/
├── cmd/server/main.go           # Application entry point & route registration
├── internal/
│   ├── config/config.go         # Configuration loading (Viper + .env)
│   ├── db/
│   │   ├── postgres.go          # PostgreSQL connection pool
│   │   └── redis.go             # Redis client
│   ├── handlers/                # HTTP request handlers (controllers)
│   │   ├── auth.go              # Register, login, logout, profile
│   │   ├── buildings.go         # Building CRUD, members, dashboard
│   │   ├── dues.go              # Dues plans, payments, reports
│   │   ├── forum.go             # Forum posts, comments, voting, media
│   │   ├── maintenance.go       # Maintenance requests, approval workflow
│   │   ├── messaging.go         # Conversations, direct messages
│   │   ├── notifications.go     # Notifications, announcements, preferences
│   │   ├── package.go           # Package tracking, pickup
│   │   ├── reservation.go       # Common areas, reservations
│   │   ├── social.go            # Follow/unfollow, user profiles
│   │   ├── timeline.go          # Community feed, posts, likes, polls
│   │   ├── visitor.go           # Visitor passes, QR check-in/out
│   │   └── ws.go                # WebSocket upgrade handler
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication & RBAC
│   │   ├── logger.go            # Request logging (Zap)
│   │   └── ratelimit.go         # Redis-backed rate limiting
│   ├── models/                  # Data structures & DTOs
│   ├── repository/              # Database queries (PostgreSQL)
│   ├── services/
│   │   ├── auth_service.go      # JWT token generation & validation
│   │   └── storage_service.go   # MinIO/S3 file uploads
│   └── websocket/
│       └── hub.go               # WebSocket hub (user connections, events)
├── migrations/                  # SQL migration files (7 migrations)
├── docs/
│   └── postman_collection.json  # Complete Postman collection
├── docker-compose.yml           # Full stack orchestration
├── Dockerfile                   # Multi-stage Go build
├── .env.example                 # Environment variable template
├── go.mod                       # Go module dependencies
└── go.sum                       # Dependency checksums
```

## Quick Start

### Prerequisites
- Docker & Docker Compose v2
- (Optional) Go 1.25+ for local development

### 1. Clone & Configure

```bash
cd apartment-backend
cp .env.example .env
# Edit .env if needed (defaults work for Docker)
```

### 2. Start All Services

```bash
docker compose up --build
```

This starts:
- **API server** at `http://localhost:8080`
- **PostgreSQL** at `localhost:5432`
- **Redis** at `localhost:6379`
- **MinIO** at `http://localhost:9000` (Console: `http://localhost:9001`)
- Runs all database migrations automatically

### 3. Verify

```bash
curl http://localhost:8080/api/v1/auth/me
# Should return 401 (no token) — server is running
```

### Local Development (without Docker)

```bash
# Start dependencies
docker compose up postgres redis minio minio-init -d

# Run migrations
docker compose up migrate

# Set local env vars
export DB_HOST=localhost REDIS_HOST=localhost MINIO_ENDPOINT=localhost:9000

# Run the server
go run cmd/server/main.go
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Environment (development/production) |
| `APP_PORT` | `8080` | Server port |
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `apartment` | Database user |
| `DB_PASSWORD` | `apartment_secret` | Database password |
| `DB_NAME` | `apartment_db` | Database name |
| `REDIS_HOST` | `redis` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_ACCESS_SECRET` | (required) | JWT access token secret |
| `JWT_REFRESH_SECRET` | (required) | JWT refresh token secret |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token TTL |
| `JWT_REFRESH_EXPIRY` | `720h` | Refresh token TTL (30 days) |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key |
| `MINIO_BUCKET` | `apartment-files` | Storage bucket name |
| `CORS_ORIGINS` | `*` | Allowed CORS origins |
| `RATE_LIMIT_MAX` | `100` | Max requests per window |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit time window |

## Database Schema

7 migration files creating 30+ tables:

### Migration 1: Core Schema
- `users` — User accounts with roles
- `buildings` — Building/apartment complexes
- `building_members` — User-building associations with roles
- `units` — Apartment units with status tracking
- `unit_residents` — Unit-to-user mapping
- `dues_plans` — Monthly/annual dues configurations
- `due_payments` — Payment records per unit
- `expenses` — Building expenses
- `maintenance_requests` — Maintenance with priority levels
- `vendors` — Service provider directory
- `notifications` — Push notifications
- `notification_preferences` — Per-user notification settings
- `forum_categories`, `forum_posts`, `forum_comments`, `forum_votes` — Building forum
- `timeline_posts`, `timeline_likes`, `timeline_comments` — Social timeline
- `polls`, `poll_options`, `poll_votes` — Poll system

### Migration 2: Invitations
- `building_invitations` — Token-based building invitation system

### Migration 3: Social Features
- `user_follows` — Follow/unfollow social graph
- Adds repost support to timeline

### Migration 4: Maintenance Approval
- Adds `pending_approval` status and manager-only approval workflow

### Migration 5: Messaging
- `conversations` — Direct and group conversations
- `conversation_participants` — Participant tracking with read receipts
- `messages` — Message storage with soft deletes

### Migration 6: Forum Media
- `forum_media` — Image attachments for forum posts

### Migration 7: Visitors, Reservations, Packages
- `visitor_passes` — QR-coded visitor passes with check-in/out
- `common_areas` — Reservable spaces with capacity/hours
- `reservations` — Booking system with overlap prevention (GIST indexes)
- `packages` — Package/parcel tracking

## API Reference

Base URL: `http://localhost:8080/api/v1`

All authenticated endpoints require: `Authorization: Bearer <access_token>`

### Authentication
| Method | Endpoint | Description |
|---|---|---|
| POST | `/auth/register` | Register new user |
| POST | `/auth/login` | Login (returns tokens) |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout (revoke refresh token) |
| POST | `/auth/accept-invitation` | Accept building invitation |
| GET | `/auth/me` | Get current user profile |
| PATCH | `/auth/me` | Update profile |
| PATCH | `/auth/password` | Change password |

### Buildings
| Method | Endpoint | Description |
|---|---|---|
| POST | `/buildings` | Create building |
| GET | `/buildings` | List user's buildings |
| GET | `/buildings/:id` | Get building details |
| GET | `/buildings/:id/dashboard` | Dashboard statistics |
| POST | `/buildings/:id/leave` | Leave a building |
| GET | `/buildings/:id/members` | List members |
| DELETE | `/buildings/:id/members/:userId` | Remove member (manager) |
| GET | `/buildings/:id/invitations` | List invitations |
| POST | `/buildings/:id/invitations` | Invite user |

### Units
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/units` | List units |
| POST | `/buildings/:id/units` | Create unit |
| PATCH | `/buildings/:id/units/:unitId` | Update unit |
| DELETE | `/buildings/:id/units/:unitId` | Delete unit |
| GET | `/buildings/:id/residents` | List residents |

### Financial — Dues
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/dues` | List dues plans |
| POST | `/buildings/:id/dues` | Create dues plan |
| PATCH | `/buildings/:id/dues/:planId` | Update plan |
| DELETE | `/buildings/:id/dues/:planId` | Delete plan |
| POST | `/buildings/:id/dues/:planId/pay` | Record payment |
| GET | `/buildings/:id/dues/report` | Financial report (query: month, year) |

### Financial — Expenses
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/expenses` | List expenses (paginated) |
| POST | `/buildings/:id/expenses` | Create expense |
| PATCH | `/buildings/:id/expenses/:expenseId` | Update expense |
| DELETE | `/buildings/:id/expenses/:expenseId` | Delete expense |

### Maintenance
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/maintenance` | List requests (paginated) |
| POST | `/buildings/:id/maintenance` | Create request |
| PATCH | `/buildings/:id/maintenance/:reqId` | Update request |
| DELETE | `/buildings/:id/maintenance/:reqId` | Delete request |
| POST | `/buildings/:id/maintenance/:reqId/approve` | Approve (manager) |
| POST | `/buildings/:id/maintenance/:reqId/reject` | Reject (manager) |

### Vendors
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/vendors` | List vendors |
| POST | `/buildings/:id/vendors` | Create vendor |
| PATCH | `/buildings/:id/vendors/:vendorId` | Update vendor |
| DELETE | `/buildings/:id/vendors/:vendorId` | Delete vendor |

### Notifications
| Method | Endpoint | Description |
|---|---|---|
| GET | `/notifications` | List notifications (paginated) |
| PATCH | `/notifications/:id/read` | Mark as read |
| POST | `/buildings/:id/announcements` | Send announcement (manager) |
| GET | `/notifications/preferences` | Get preferences |
| PATCH | `/notifications/preferences` | Update preferences |

### Forum
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/forum/categories` | List categories |
| GET | `/buildings/:id/forum/posts` | List posts (paginated, filterable) |
| POST | `/buildings/:id/forum/posts` | Create post |
| GET | `/buildings/:id/forum/posts/:postId` | Get post detail |
| POST | `/buildings/:id/forum/posts/:postId/comments` | Add comment |
| POST | `/buildings/:id/forum/posts/:postId/vote` | Vote (+1/-1) |
| POST | `/buildings/:id/forum/posts/:postId/media` | Upload media (multipart) |

### Timeline (Community Feed)
| Method | Endpoint | Description |
|---|---|---|
| GET | `/timeline` | Get feed (paginated) |
| POST | `/timeline` | Create post (text/poll/location) |
| GET | `/timeline/:postId` | Get single post |
| POST | `/timeline/:postId/like` | Toggle like |
| GET | `/timeline/:postId/comments` | Get comments |
| POST | `/timeline/:postId/comments` | Add comment |
| POST | `/timeline/:postId/repost` | Repost |
| DELETE | `/timeline/:postId/repost` | Unrepost |
| POST | `/timeline/polls/:pollId/vote` | Vote on poll |
| GET | `/timeline/nearby` | Nearby posts (query: lat, lng, radius) |

### Social
| Method | Endpoint | Description |
|---|---|---|
| GET | `/users/search` | Search users (query: q) |
| GET | `/users/:id/profile` | User profile + follow status |
| POST | `/users/:id/follow` | Follow user |
| DELETE | `/users/:id/follow` | Unfollow user |
| GET | `/users/:id/followers` | List followers (paginated) |
| GET | `/users/:id/following` | List following (paginated) |

### Messaging
| Method | Endpoint | Description |
|---|---|---|
| GET | `/messages/conversations` | List conversations |
| POST | `/messages/conversations` | Start/get conversation (body: user_id) |
| GET | `/messages/conversations/:convId/messages` | Get messages (cursor: before) |
| POST | `/messages/conversations/:convId/messages` | Send message |
| POST | `/messages/conversations/:convId/read` | Mark as read |

### Visitors
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/visitors` | List visitor passes |
| POST | `/buildings/:id/visitors` | Create pass (returns QR) |
| POST | `/buildings/:id/visitors/:passId/checkin` | Check in visitor |
| POST | `/buildings/:id/visitors/:passId/checkout` | Check out visitor |
| DELETE | `/buildings/:id/visitors/:passId` | Cancel pass |
| GET | `/buildings/:id/visitors/scan/:qr` | Scan QR code |

### Common Areas & Reservations
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/areas` | List common areas |
| POST | `/buildings/:id/areas` | Create area (manager) |
| GET | `/buildings/:id/reservations` | List reservations (query: area_id, date) |
| POST | `/buildings/:id/reservations` | Create reservation |
| POST | `/buildings/:id/reservations/:resId/approve` | Approve (manager) |
| POST | `/buildings/:id/reservations/:resId/reject` | Reject (manager) |
| DELETE | `/buildings/:id/reservations/:resId` | Cancel reservation |
| GET | `/reservations/my` | User's reservations |

### Packages
| Method | Endpoint | Description |
|---|---|---|
| GET | `/buildings/:id/packages` | List packages (query: status) |
| POST | `/buildings/:id/packages` | Log package arrival |
| POST | `/buildings/:id/packages/:pkgId/pickup` | Mark picked up |
| POST | `/buildings/:id/packages/:pkgId/notify` | Notify recipient |
| GET | `/packages/my` | User's packages |

### WebSocket
| Protocol | Endpoint | Description |
|---|---|---|
| WS | `/ws?token=ACCESS_TOKEN` | Real-time event stream |

**WebSocket Events:**
- `new_message` — New direct message received
- `new_notification` — New notification (like, comment, follow, repost, announcement)

## Authentication & Authorization

### JWT Token Flow
1. **Register/Login** → Returns `access_token` (15min) + `refresh_token` (30 days)
2. **API Requests** → Include `Authorization: Bearer <access_token>`
3. **Token Expired** → Call `/auth/refresh` with refresh token
4. **Logout** → Revokes refresh token

### Role-Based Access Control (RBAC)
| Role | Permissions |
|---|---|
| `resident` | View building data, create requests, participate in forum/timeline |
| `building_manager` | All resident permissions + manage units, dues, approve requests, create announcements, manage areas |

Manager-only endpoints are protected by `managerOnly` middleware.

## Response Format

All endpoints return a consistent JSON structure:

```json
// Success
{
  "success": true,
  "data": { ... },
  "message": "Operation completed"
}

// Error
{
  "success": false,
  "error": "Error description"
}

// Paginated
{
  "success": true,
  "data": {
    "items": [...],
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

## Postman Collection

Import `docs/postman_collection.json` into Postman. Collection features:
- Auto-sets tokens after login/register
- Auto-captures created resource IDs
- Collection variables for all entity IDs
- 80+ requests covering all endpoints

## Docker Services

| Service | Port | Description |
|---|---|---|
| `app` | 8080 | Go API server |
| `postgres` | 5432 | PostgreSQL 16 |
| `redis` | 6379 | Redis 7 |
| `minio` | 9000/9001 | MinIO (API/Console) |
| `migrate` | — | Runs DB migrations on startup |
| `minio-init` | — | Creates storage bucket |
