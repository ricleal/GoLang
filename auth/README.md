# Authentication & Authorization Proof of Concept

This project demonstrates a microservices architecture with **decoupled authentication and authorization** using JWT tokens.

## Architecture

```
┌─────────────────┐
│   API Gateway   │ :8000  (Entry point)
└────────┬────────┘
         │
    ┌────┴─────────────────────────────┐
    │                                  │
┌───▼─────┐                       ┌───▼────┐
│   App   │ :8081 (Replica 1)     │  Auth  │
├─────────┤                       │ Server │
│   App   │ :8081 (Replica 2)     │ :8080  │
└─────────┘                       └────────┘
    │                                  │
    └──────────────┬───────────────────┘
                   │
             (Validates JWT)

Docker DNS round-robin load balancing across app replicas
```

### Components

1. **Auth Server** (`:8080`)
   - Issues JWT tokens upon successful login
   - Validates JWT tokens for other services
   - Decoupled from business logic

2. **API Gateway** (`:8000`)
   - Entry point for all client requests
   - Validates JWT with auth server before routing
   - Proxies to app service (Docker DNS handles load balancing)

3. **App Service** (`:8081` - 2 replicas)
   - Business logic services (cowsay implementation)
   - Scaled with Docker Compose replicas (horizontally scalable)
   - Each validates JWT with auth server
   - Completely decoupled from authentication logic
   - Docker DNS provides automatic round-robin load balancing

## Key Features

✅ **Decoupled Auth**: Authentication logic is completely separated from business services  
✅ **JWT-based**: Stateless authentication using JWT tokens  
✅ **Role-Based Access Control (RBAC)**: Users have roles (user, admin) with different permissions  
✅ **Horizontal Scaling**: App service uses Docker Compose replicas for easy scaling  
✅ **Load Balancing**: Round-robin distribution across multiple app instances  
✅ **Health Checks**: All services expose health endpoints  
✅ **Containerized**: Fully containerized with Docker Compose  

## Quick Start

### Prerequisites

- Docker and Docker Compose installed
- curl or any HTTP client

### 1. Start all services

```bash
cd auth
docker-compose up --build
```

Wait for all services to be healthy (~30 seconds).

### 2. Login to get JWT token

```bash
# Login as alice
curl -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-12-30T15:30:00Z"
}
```

Save the token for subsequent requests.

### 3. Call the protected API

```bash
# Set your token
TOKEN="<your-token-from-login>"

# Make cowsay request
curl -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello from Go!"}'
```

Response:
```json
{
  "cow": " ---------------- \n< Hello from Go! >\n ---------------- \n        \\   ^__^\n         \\  (oo)\\_______\n            (__)\\       )\\/\\\n                ||----w |\n                ||     ||\n",
  "message": "Hello from Go!",
  "service": "app1",
  "user": "alice"
}
```

Notice the `service` field alternates between `app1` and `app2` due to load balancing!

## Available Users

The auth server has the following test users with different roles:

| Username | Password | Role | Permissions |
|----------|----------|------|-------------|
| **alice** | password123 | user | Can access `/api/v1/cowsay` |
| **bob** | password456 | user | Can access `/api/v1/cowsay` |
| **admin** | admin123 | admin | Can access all endpoints including `/api/v1/admin` |

## Role-Based Access Control (RBAC)

The system implements role-based access control where different users have different permissions:

### Testing RBAC

```bash
# Run the comprehensive RBAC test
./test-rbac.sh
```

### Manual RBAC Testing

```bash
# Login as admin
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# Access admin endpoint (should work)
curl -X GET http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.'

# Login as regular user
USER_TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' | jq -r '.token')

# Try to access admin endpoint (should fail with 403)
curl -X GET http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $USER_TOKEN"
```

## API Endpoints

### API Gateway (`:8000`)

| Endpoint | Method | Auth Required | Role Required | Description |
|----------|--------|---------------|---------------|-------------|
| `/login` | POST | No | - | Get JWT token |
| `/api/v1/cowsay` | POST | Yes | any | Cowsay service (load balanced) |
| `/api/v1/admin` | GET | Yes | admin | Admin panel (admin only) |
| `/health` | GET | No | - | Health check |
| `/info` | GET | No | - | Gateway information |

### Auth Server (`:8080`) - Internal

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/login` | POST | Issue JWT token |
| `/validate` | POST | Validate token (body) |
| `/validate-header` | GET | Validate token (header) |
| `/health` | GET | Health check |

### App1/App2 (`:8081`, `:8082`) - Internal

| Endpoint | Method | Auth Required | Role Required | Description |
|----------|--------|---------------|---------------|-------------|
| `/api/v1/cowsay` | POST | Yes | any | Generate cowsay |
| `/api/v1/admin` | GET | Yes | admin | Admin panel |
| `/health` | GET | No | - | Health check |
| `/info` | GET | No | - | Service info |

## Testing Script

```bash
#!/bin/bash

echo "=== Auth & Authorization PoC Test ==="
echo ""

# 1. Login
echo "1. Logging in as alice..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to get token"
  exit 1
fi

echo "✅ Token obtained: ${TOKEN:0:50}..."
echo ""

# 2. Test without auth (should fail)
echo "2. Testing without authentication..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Content-Type: application/json" \
  -d '{"message":"Test"}')
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ]; then
  echo "✅ Correctly rejected (401 Unauthorized)"
else
  echo "❌ Should have been rejected"
fi
echo ""

# 3. Test with valid auth
echo "3. Testing with valid authentication..."
for i in {1..4}; do
  echo "   Request $i:"
  curl -s -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Request $i\"}" | grep -o '"service":"[^"]*' | cut -d'"' -f4
done
echo ""
echo "✅ Notice the load balancing between app1 and app2!"
echo ""

# 4. Test with invalid token
echo "4. Testing with invalid token..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer invalid-token-here" \
  -H "Content-Type: application/json" \
  -d '{"message":"Test"}')
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ]; then
  echo "✅ Correctly rejected (401 Unauthorized)"
else
  echo "❌ Should have been rejected"
fi

echo ""
echo "=== All tests completed ==="
```

Save this as `test.sh`, make it executable (`chmod +x test.sh`), and run it!

## Architecture Decisions

### Why API Gateway?

1. **Single Entry Point**: Clients only need to know one URL
2. **Authentication at the Edge**: Validate tokens before routing to services
3. **Load Balancing**: Distribute traffic across multiple service instances
4. **Service Discovery**: Backend services can scale independently

### Why Separate Auth Server?

1. **Separation of Concerns**: Auth logic is isolated from business logic
2. **Reusability**: Multiple services can use the same auth server
3. **Scalability**: Can scale auth independently based on demand
4. **Security**: Centralized security management

### Security Considerations

⚠️ **This is a PoC - Not production ready!**

For production, consider:

- Use environment variables for JWT secret (not hardcoded)
- Implement token refresh mechanism
- Use HTTPS/TLS for all communication
- Add rate limiting
- Implement proper logging and monitoring
- Use a database for user management (not in-memory)
- Add CORS configuration
- Implement token revocation/blacklisting
- Use more sophisticated load balancing (e.g., Nginx, Traefik)

## Monitoring

Check service health:

```bash
# API Gateway
curl http://localhost:8000/health
curl http://localhost:8000/info

# Auth Server
curl http://localhost:8080/health

# App1
curl http://localhost:8081/health
curl http://localhost:8081/info

# App2
curl http://localhost:8082/health
curl http://localhost:8082/info
```

## Logs

View logs from all services:

```bash
docker-compose logs -f
```

View logs from specific service:

```bash
docker-compose logs -f api-gateway
docker-compose logs -f auth-server
docker-compose logs -f app1
docker-compose logs -f app2
```

## Stopping the Services

```bash
docker-compose down
```

Clean up (remove volumes):

```bash
docker-compose down -v
```

## Project Structure

```
auth/
├── docker-compose.yml          # Orchestrates all services
├── README.md                   # This file
├── test.sh                     # Testing script
├── auth-server/
│   ├── main.go                 # JWT issuer & validator
│   ├── go.mod
│   └── Dockerfile
├── api-gateway/
│   ├── main.go                 # Load balancer & auth proxy
│   ├── go.mod
│   └── Dockerfile
├── app1/
│   ├── main.go                 # Cowsay service instance 1
│   ├── go.mod
│   └── Dockerfile
└── app2/
    ├── main.go                 # Cowsay service instance 2
    ├── go.mod
    └── Dockerfile
```

## Further Enhancements

- [ ] Add database for user management
- [ ] Implement token refresh mechanism
- [ ] Add role-based access control (RBAC)
- [ ] Implement circuit breaker pattern
- [ ] Add distributed tracing (OpenTelemetry)
- [ ] Add metrics (Prometheus)
- [ ] Add more sophisticated load balancing algorithms
- [ ] Implement service mesh (Istio, Linkerd)

## License

MIT License - Feel free to use this as a learning resource!
