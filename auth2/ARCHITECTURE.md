# Architecture Comparison: auth vs auth2

This document compares the two authentication/authorization architectures in this repository.

## High-Level Comparison

### auth (Custom API Gateway)
```
Client → Traefik → Custom API Gateway → Auth Server → App
                    (validates JWT)
```

### auth2 (Envoy Gateway)
```
Client → Traefik → Envoy Gateway → Auth Server → App
                   (ext_authz filter)
```

## Detailed Comparison

| Aspect | auth (Custom) | auth2 (Envoy) |
|--------|--------------|---------------|
| **Gateway** | Go application | Envoy Proxy |
| **Auth Method** | Gateway validates JWT | Envoy ext_authz → Auth Server |
| **Complexity** | Simple, easy to understand | More sophisticated |
| **Maintenance** | Custom code to maintain | Community-maintained |
| **Performance** | Good (Go) | Excellent (C++) |
| **Features** | Basic reverse proxy | Advanced routing, retries, circuit breakers |
| **Observability** | Custom logging | Built-in metrics, tracing |
| **Load Balancing** | Docker DNS round-robin | Envoy's advanced LB algorithms |
| **Health Checks** | Docker Compose | Envoy + Docker Compose |
| **Production Readiness** | Proof of concept | Battle-tested |

## Component Comparison

### 1. Gateway Layer

#### auth - Custom API Gateway (Go)
**File**: [auth/api-gateway/main.go](../auth/api-gateway/main.go)

**Responsibilities**:
- HTTP reverse proxy
- JWT validation (calls auth-server)
- Header enrichment (X-Username, X-Role)
- Request routing

**Pros**:
- Full control over logic
- Easy to modify
- Simple to understand

**Cons**:
- Need to implement features manually
- Less battle-tested
- Limited observability

#### auth2 - Envoy Gateway
**File**: [auth2/envoy.yaml](envoy.yaml)

**Responsibilities**:
- HTTP reverse proxy
- External auth (ext_authz filter)
- Header manipulation
- Advanced routing & load balancing
- Health checking
- Metrics & observability

**Pros**:
- Production-grade
- Rich feature set
- Excellent performance
- Strong observability
- Battle-tested

**Cons**:
- More complex configuration
- YAML-based config
- Learning curve

### 2. Auth Server

#### auth - Auth Server (Go)
**File**: [auth/auth-server/main.go](../auth/auth-server/main.go)

**Endpoints**:
- `/login` - Issue JWT
- `/validate` - Validate JWT (body)
- `/validate-header` - Validate JWT (header)

**Used by**: API Gateway

#### auth2 - Auth Server (Go) with Envoy Integration
**File**: [auth2/auth-server/main.go](auth-server/main.go)

**Endpoints**:
- `/login` - Issue JWT
- `/auth` - Envoy external auth endpoint

**Used by**: Envoy Gateway (ext_authz filter)

**Key Difference**: The auth2 version follows Envoy's external auth protocol:
- Returns 200 OK with headers if valid
- Returns 401/403 for invalid tokens
- Headers added: `X-Username`, `X-Role`

### 3. App Service

Both versions are nearly identical:
- [auth/app/main.go](../auth/app/main.go)
- [auth2/app/main.go](app/main.go)

**Difference**: Comments reference "API Gateway" vs "Envoy Gateway"

### 4. Entry Point

Both use **Traefik** as the entry point:
- Single exposed port (8000)
- Dashboard on 8080
- Load balancing across gateway replicas

## Authentication Flow Comparison

### auth Flow
```
1. Client → Traefik (:8000) with JWT
2. Traefik → API Gateway (round-robin)
3. API Gateway → Auth Server (/validate-header)
4. Auth Server validates & returns {valid, username, role}
5. API Gateway adds X-Username, X-Role headers
6. API Gateway → App (Docker DNS load balancing)
7. App checks headers & performs authorization
8. Response flows back
```

### auth2 Flow
```
1. Client → Traefik (:8000) with JWT
2. Traefik → Envoy Gateway
3. Envoy ext_authz filter → Auth Server (/auth)
4. Auth Server validates & returns 200 + headers or 401
5. Envoy adds headers from auth response
6. Envoy → App (Envoy load balancing)
7. App checks headers & performs authorization
8. Response flows back
```

## When to Use Which?

### Use auth (Custom Gateway) when:
- ✅ Learning/educational purposes
- ✅ Simple use case
- ✅ Full control over logic required
- ✅ Small team without Envoy experience
- ✅ Rapid prototyping

### Use auth2 (Envoy Gateway) when:
- ✅ Production deployment
- ✅ Need advanced features (retries, circuit breakers, etc.)
- ✅ Require strong observability
- ✅ High performance requirements
- ✅ Team has or wants to learn Envoy
- ✅ Multiple services/microservices
- ✅ Need service mesh capabilities

## Migration Path

To migrate from **auth** to **auth2**:

1. **Keep existing services**: Auth Server and App need minimal changes
2. **Replace API Gateway**: Swap custom gateway with Envoy
3. **Update Auth Server**: Add Envoy-compatible `/auth` endpoint
4. **Configure Envoy**: Set up ext_authz filter
5. **Test thoroughly**: Ensure JWT validation works
6. **Monitor**: Use Envoy metrics for observability

## Performance Comparison

### Theoretical Performance
- **auth**: Go runtime, ~50-100k req/s per instance
- **auth2**: C++ (Envoy), ~100-200k req/s per instance

### Memory Usage
- **auth**: 10-30 MB per gateway instance
- **auth2**: 30-50 MB per Envoy instance

*Note: Actual performance depends on hardware, load, and configuration.*

## Observability Comparison

### auth
- Docker Compose logs
- Application logs
- Traefik dashboard

### auth2
- Docker Compose logs
- Application logs
- Traefik dashboard
- **Envoy admin interface** (:9901)
- **Envoy metrics** (Prometheus-compatible)
- **Envoy access logs**
- **Envoy stats endpoint**

## Configuration Complexity

### auth
**Configuration files**: 3 main files
- `docker-compose.yml` - Service orchestration
- `api-gateway/main.go` - Gateway logic
- `auth-server/main.go` - Auth logic

**Lines of configuration**: ~300 lines

### auth2
**Configuration files**: 4 main files
- `docker-compose.yml` - Service orchestration
- `envoy.yaml` - Envoy configuration
- `auth-server/main.go` - Auth logic (with Envoy endpoint)
- `app/main.go` - App logic

**Lines of configuration**: ~400 lines

**Complexity**: auth2 is ~30% more complex but provides 10x more features.

## Conclusion

Both architectures are valid for different use cases:

- **auth** is great for **learning, prototyping, and simple deployments**
- **auth2** is better for **production, scalability, and advanced requirements**

The investment in learning Envoy pays off for production systems, especially as you scale to multiple services and need advanced features like:
- Circuit breakers
- Retry policies
- Advanced load balancing
- Distributed tracing
- gRPC support
- Service mesh integration

## Further Reading

- [Envoy Documentation](https://www.envoyproxy.io/docs)
- [Envoy ext_authz Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter)
- [Traefik Documentation](https://doc.traefik.io/traefik/)
