#!/bin/bash

echo "🚦 Testing Traefik Rate Limiting"
echo "================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Login and get token
echo -e "${YELLOW}1. Getting authentication token...${NC}"
TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo -e "${RED}❌ Failed to get token${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Token obtained${NC}"
echo ""

# Test normal requests (should succeed)
echo -e "${YELLOW}2. Testing normal request rate (should succeed)...${NC}"
SUCCESS=0
for i in {1..10}; do
  RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Request $i\"}")
  
  STATUS=$(echo "$RESPONSE" | tail -n1)
  if [ "$STATUS" = "200" ]; then
    ((SUCCESS++))
  fi
  sleep 0.05  # 50ms between requests = 20 req/sec (well under limit)
done

echo -e "${GREEN}✅ Normal rate: $SUCCESS/10 succeeded${NC}"
echo ""

# Test rate limiting (send many requests quickly)
echo -e "${YELLOW}3. Testing rate limiting (sending 200 requests rapidly)...${NC}"
echo "   Rate limit: 10 req/sec average, burst of 20"
echo ""

SUCCESS=0
RATE_LIMITED=0

for i in {1..200}; do
  STATUS=$(curl -s -w "%{http_code}" -o /dev/null -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Burst test $i\"}")
  
  if [ "$STATUS" = "200" ]; then
    ((SUCCESS++))
  elif [ "$STATUS" = "429" ]; then
    ((RATE_LIMITED++))
  fi
  
  # Show progress
  if [ $((i % 20)) -eq 0 ]; then
    echo "   Sent $i requests... (Success: $SUCCESS, Rate Limited: $RATE_LIMITED)"
  fi
done

echo ""
echo -e "${YELLOW}Results:${NC}"
echo -e "  ${GREEN}✅ Successful requests: $SUCCESS${NC}"
echo -e "  ${RED}🚫 Rate limited (429): $RATE_LIMITED${NC}"
echo ""

if [ $RATE_LIMITED -gt 0 ]; then
  echo -e "${GREEN}✅ Rate limiting is working!${NC}"
  echo "   Traefik blocked excessive requests with HTTP 429"
else
  echo -e "${YELLOW}⚠️  No rate limiting detected${NC}"
  echo "   This might mean the rate limit is too high for this test"
fi

echo ""
echo -e "${YELLOW}4. Waiting 2 seconds and testing recovery...${NC}"
sleep 2

RECOVERY_SUCCESS=0
for i in {1..5}; do
  STATUS=$(curl -s -w "%{http_code}" -o /dev/null -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Recovery test $i\"}")
  
  if [ "$STATUS" = "200" ]; then
    ((RECOVERY_SUCCESS++))
  fi
  sleep 0.2
done

echo -e "${GREEN}✅ Recovery test: $RECOVERY_SUCCESS/5 succeeded${NC}"
echo ""

# Check Traefik dashboard
echo -e "${YELLOW}5. Traefik Dashboard:${NC}"
echo "   URL: http://localhost:8080/dashboard/"
echo "   View load balancing and service health"
echo ""

echo -e "${GREEN}================================="
echo "Rate limiting test complete!"
echo -e "=================================${NC}"
