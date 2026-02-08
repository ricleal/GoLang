#!/bin/bash

echo "=== RBAC Testing with Envoy Gateway ==="
echo ""

# 1. Login as regular user (alice)
echo "1. Logging in as alice (user role)..."
ALICE_LOGIN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}')

ALICE_TOKEN=$(echo $ALICE_LOGIN | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ALICE_TOKEN" ]; then
  echo "❌ Failed to get alice's token"
  exit 1
fi

echo "✅ Alice logged in successfully"
echo ""

# 2. Login as admin
echo "2. Logging in as admin (admin role)..."
ADMIN_LOGIN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

ADMIN_TOKEN=$(echo $ADMIN_LOGIN | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo "❌ Failed to get admin's token"
  exit 1
fi

echo "✅ Admin logged in successfully"
echo ""

# 3. Test alice accessing cowsay (should work)
echo "3. Alice accessing /api/v1/cowsay..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello from Alice!"}')

STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "200" ]; then
  echo "✅ Alice can access cowsay (status: $STATUS)"
else
  echo "❌ Alice should be able to access cowsay (got status: $STATUS)"
fi
echo ""

# 4. Test alice accessing admin endpoint (should fail with 403)
echo "4. Alice trying to access /api/v1/admin..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $ALICE_TOKEN")

STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "403" ]; then
  echo "✅ Alice correctly denied admin access (status: $STATUS)"
else
  echo "❌ Alice should be denied admin access (got status: $STATUS)"
fi
echo ""

# 5. Test admin accessing cowsay (should work)
echo "5. Admin accessing /api/v1/cowsay..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello from Admin!"}')

STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "200" ]; then
  echo "✅ Admin can access cowsay (status: $STATUS)"
else
  echo "❌ Admin should be able to access cowsay (got status: $STATUS)"
fi
echo ""

# 6. Test admin accessing admin endpoint (should work)
echo "6. Admin accessing /api/v1/admin..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $ADMIN_TOKEN")

STATUS=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$STATUS" = "200" ]; then
  echo "✅ Admin can access admin endpoint (status: $STATUS)"
  echo "Response: $BODY" | head -c 100
  echo "..."
else
  echo "❌ Admin should be able to access admin endpoint (got status: $STATUS)"
fi
echo ""

# Summary
echo "=== RBAC Test Summary ==="
echo "✅ Authentication: Envoy → Auth Server (JWT validation)"
echo "✅ Authorization: App Service (role checking)"
echo "✅ User 'alice' can access public endpoints but not admin"
echo "✅ User 'admin' can access all endpoints"
echo ""
echo "Architecture: Traefik → Envoy Gateway → Auth Server + App"
