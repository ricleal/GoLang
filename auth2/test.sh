#!/bin/bash

echo "=== Auth & Authorization with Envoy Gateway Test ==="
echo ""

# 1. Login
echo "1. Logging in as alice..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to get token"
  echo "Response: $LOGIN_RESPONSE"
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
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
  echo "✅ Correctly rejected ($STATUS Unauthorized)"
else
  echo "❌ Should have been rejected (got $STATUS)"
fi
echo ""

# 3. Test with valid auth
echo "3. Testing with valid authentication (4 requests to see load balancing)..."
for i in {1..4}; do
  echo "   Request $i:"
  RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/cowsay \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Request $i\"}")
  
  INSTANCE=$(echo "$RESPONSE" | grep -o '"instance":"[^"]*' | cut -d'"' -f4)
  if [ -n "$INSTANCE" ]; then
    echo "   ✅ Served by instance: $INSTANCE"
  else
    echo "   ❌ Unexpected response: $RESPONSE"
  fi
done
echo ""
echo "✅ Notice Envoy load balancing between service replicas!"
echo ""

# 4. Test with invalid token
echo "4. Testing with invalid token..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer invalid-token-here" \
  -H "Content-Type: application/json" \
  -d '{"message":"Test"}')
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
  echo "✅ Correctly rejected ($STATUS Unauthorized)"
else
  echo "❌ Should have been rejected (got $STATUS)"
fi

echo ""
echo "=== All tests completed ==="
echo ""
echo "💡 Tip: Check Envoy stats with:"
echo "   docker exec -it auth2-envoy wget -q -O- http://localhost:9901/stats | grep auth"
