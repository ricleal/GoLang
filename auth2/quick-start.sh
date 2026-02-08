#!/bin/bash

echo "=============================================="
echo "  Auth2 - Envoy Gateway Quick Start"
echo "=============================================="
echo ""

# Check if docker is running
if ! docker info > /dev/null 2>&1; then
  echo "❌ Docker is not running. Please start Docker first."
  exit 1
fi

echo "📦 Starting services with docker compose..."
echo ""
docker compose up --build -d

echo ""
echo "⏳ Waiting for services to be healthy (this may take 30-60 seconds)..."
echo ""

# Wait for services to be healthy
MAX_WAIT=60
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
  HEALTHY=$(docker compose ps | grep -c "healthy")
  TOTAL=$(docker compose ps | grep -c "Up")
  
  if [ $HEALTHY -ge 3 ]; then
    echo "✅ All services are healthy!"
    break
  fi
  
  echo "   Waiting... ($HEALTHY/$TOTAL services healthy)"
  sleep 5
  ELAPSED=$((ELAPSED + 5))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
  echo "⚠️  Timeout waiting for services. Check status with: docker compose ps"
  echo ""
  docker compose ps
  exit 1
fi

echo ""
echo "=============================================="
echo "  Services are ready!"
echo "=============================================="
echo ""
echo "📊 Traefik Dashboard: http://localhost:8080/dashboard/"
echo ""
echo "Try these commands:"
echo ""
echo "1. Login and get token:"
echo '   curl -X POST http://localhost:8000/login \'
echo '     -H "Content-Type: application/json" \'
echo '     -d '"'"'{"username":"alice","password":"password123"}'"'"
echo ""
echo "2. Run test suite:"
echo "   ./test.sh"
echo ""
echo "3. Test RBAC:"
echo "   ./test-rbac.sh"
echo ""
echo "4. View logs:"
echo "   docker compose logs -f"
echo ""
echo "5. Stop services:"
echo "   docker compose down"
echo ""
echo "=============================================="
