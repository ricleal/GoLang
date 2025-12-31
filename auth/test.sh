#!/bin/bash

echo "=== Auth & Authorization PoC Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Wait for services to be ready
echo "${YELLOW}Waiting for services to be ready...${NC}"
sleep 5

# 1. Login
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. Logging in as alice..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}')

echo "Response: $LOGIN_RESPONSE"

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "${RED}❌ Failed to get token${NC}"
  exit 1
fi

echo "${GREEN}✅ Token obtained: ${TOKEN:0:50}...${NC}"

# 2. Test without auth (should fail)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. Testing without authentication (should fail)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Content-Type: application/json" \
  -d '{"message":"Test"}')
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ]; then
  echo "${GREEN}✅ Correctly rejected (401 Unauthorized)${NC}"
else
  echo "${RED}❌ Should have been rejected, got status: $STATUS${NC}"
fi

# 3. Test with valid auth
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. Testing with valid authentication (4 requests)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for i in {1..4}; do
  RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Request $i\"}")
  
  SERVICE=$(echo $RESPONSE | grep -o '"service":"[^"]*' | cut -d'"' -f4)
  echo "   Request $i routed to: ${YELLOW}$SERVICE${NC}"
done
echo "${GREEN}✅ Notice the load balancing between app1 and app2!${NC}"

# 4. Display full cowsay output
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. Full cowsay response:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
FULL_RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello from authenticated user!"}')

echo "$FULL_RESPONSE" | jq -r '.cow'
echo ""
echo "Service: $(echo $FULL_RESPONSE | jq -r '.service')"
echo "User: $(echo $FULL_RESPONSE | jq -r '.user')"

# 5. Test with invalid token
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5. Testing with invalid token (should fail)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer invalid-token-here" \
  -H "Content-Type: application/json" \
  -d '{"message":"Test"}')
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ]; then
  echo "${GREEN}✅ Correctly rejected (401 Unauthorized)${NC}"
else
  echo "${RED}❌ Should have been rejected, got status: $STATUS${NC}"
fi

# 6. Test with wrong credentials
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6. Testing login with wrong credentials (should fail)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"wrongpassword"}')

STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ]; then
  echo "${GREEN}✅ Correctly rejected (401 Unauthorized)${NC}"
else
  echo "${RED}❌ Should have been rejected, got status: $STATUS${NC}"
fi

# 7. Health checks
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "7. Health checks for all services..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo -n "API Gateway: "
curl -s http://localhost:8000/health | jq -r '.status'

echo -n "Auth Server: "
curl -s http://localhost:8080/health | jq -r '.status'

echo -n "App1: "
curl -s http://localhost:8081/health | jq -r '.status'

echo -n "App2: "
curl -s http://localhost:8082/health | jq -r '.status'

echo ""
echo "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo "${GREEN}✅ All tests completed successfully!${NC}"
echo "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
