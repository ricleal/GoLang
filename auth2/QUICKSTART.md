# Quick Start Guide - auth2 (Envoy Gateway)

Get up and running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- curl or any HTTP client
- (Optional) jq for JSON formatting

## 1. Start Everything

```bash
cd auth2
./quick-start.sh
```

Or manually:

```bash
docker compose up --build
```

Wait ~30 seconds for services to be healthy.

## 2. Login

```bash
# Login as alice (user role)
curl -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' | jq '.'
```

Copy the `token` from the response.

## 3. Make a Request

```bash
# Replace YOUR_TOKEN with the actual token
TOKEN="YOUR_TOKEN"

curl -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello Envoy!"}' | jq '.'
```

## 4. Run Tests

```bash
# Basic functionality test
./test.sh

# RBAC test (user vs admin)
./test-rbac.sh
```

## Test Users

| Username | Password | Role | Access |
|----------|----------|------|--------|
| alice | password123 | user | /api/v1/cowsay |
| bob | password456 | user | /api/v1/cowsay |
| admin | admin123 | admin | All endpoints |

## Useful Commands

```bash
# View logs
docker compose logs -f

# View logs from specific service
docker compose logs -f envoy
docker compose logs -f auth-server
docker compose logs -f app

# Check service status
docker compose ps

# View Envoy stats
docker exec -it auth2-envoy wget -q -O- http://localhost:9901/stats | head -20

# Stop everything
docker compose down

# Clean restart
docker compose down -v && docker compose up --build
```

## Endpoints

| Endpoint | Method | Auth? | Description |
|----------|--------|-------|-------------|
| `/login` | POST | No | Get JWT token |
| `/api/v1/cowsay` | POST | Yes | Cowsay service |
| `/api/v1/admin` | GET | Yes (admin only) | Admin panel |
| `/health` | GET | No | Health check |

## Dashboards

- **Traefik**: http://localhost:8080/dashboard/
- **Envoy Admin**: `docker exec -it auth2-envoy wget -q -O- http://localhost:9901/`

## Architecture

```
Client 
  ↓
Traefik (:8000) - Entry point
  ↓
Envoy Gateway (:10000) - Auth & routing
  ↓ (External Auth)
Auth Server (:8080) - JWT validation
  ↓
App Service (:8081) - Business logic (2 replicas)
```

## What's Happening?

1. **Traefik** receives all requests on port 8000
2. **Envoy** intercepts protected routes and calls auth-server
3. **Auth Server** validates JWT and returns user info
4. **Envoy** adds X-Username and X-Role headers
5. **App** receives request with headers and performs authorization
6. **Envoy** load balances across app replicas

## Troubleshooting

### Services not starting?
```bash
docker compose down -v
docker compose up --build
```

### Can't get token?
Check if all services are healthy:
```bash
docker compose ps
```

### Token not working?
- Check token didn't expire (15 min lifetime)
- Ensure you're using `Bearer TOKEN` format
- Check logs: `docker compose logs auth-server`

### Connection refused?
Wait for services to be healthy (~30 seconds after start).

## Next Steps

- Read [README.md](README.md) for detailed architecture
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for comparison with auth
- Explore Envoy configuration in [envoy.yaml](envoy.yaml)
- Try scaling: `docker compose up --scale app=5`

## Stop & Clean Up

```bash
# Stop services
docker compose down

# Clean up everything (including volumes)
docker compose down -v

# Remove images too
docker compose down -v --rmi all
```

---

**Need help?** Check the full [README.md](README.md) or [ARCHITECTURE.md](ARCHITECTURE.md)
