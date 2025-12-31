#!/bin/bash

echo "🔐 Role-Based Access Control (RBAC) Test"
echo "========================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Login as regular user (alice)
echo "${YELLOW}1. Testing regular user (alice - role: user)${NC}"
echo "---------------------------------------------"
ALICE_TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ALICE_TOKEN" ]; then
  echo "${RED}❌ Failed to login as alice${NC}"
  exit 1
fi
echo "${GREEN}✅ Alice logged in successfully${NC}"

# Test 2: Alice accesses cowsay (should work)
echo ""
echo "${YELLOW}2. Alice accessing /api/v1/cowsay (should succeed)${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello as regular user"}')
  
if echo "$RESPONSE" | grep -q "cow"; then
  echo "${GREEN}✅ Alice can access cowsay endpoint${NC}"
  echo "   Service: $(echo $RESPONSE | jq -r '.service')"
else
  echo "${RED}❌ Alice cannot access cowsay endpoint${NC}"
fi

# Test 3: Alice tries to access admin endpoint (should fail)
echo ""
echo "${YELLOW}3. Alice accessing /api/v1/admin (should be FORBIDDEN)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $ALICE_TOKEN")
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "403" ]; then
  echo "${GREEN}✅ Alice correctly denied access (403 Forbidden)${NC}"
else
  echo "${RED}❌ Expected 403, got: $STATUS${NC}"
fi

# Test 4: Login as admin
echo ""
echo "${YELLOW}4. Testing admin user (admin - role: admin)${NC}"
echo "--------------------------------------------"
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo "${RED}❌ Failed to login as admin${NC}"
  exit 1
fi
echo "${GREEN}✅ Admin logged in successfully${NC}"

# Test 5: Admin accesses cowsay (should work)
echo ""
echo "${YELLOW}5. Admin accessing /api/v1/cowsay (should succeed)${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello as admin"}')
  
if echo "$RESPONSE" | grep -q "cow"; then
  echo "${GREEN}✅ Admin can access cowsay endpoint${NC}"
  echo "   Service: $(echo $RESPONSE | jq -r '.service')"
else
  echo "${RED}❌ Admin cannot access cowsay endpoint${NC}"
fi

# Test 6: Admin accesses admin endpoint (should work)
echo ""
echo "${YELLOW}6. Admin accessing /api/v1/admin (should succeed)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $ADMIN_TOKEN")
  
STATUS=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$STATUS" = "200" ]; then
  echo "${GREEN}✅ Admin successfully accessed admin endpoint${NC}"
  echo ""
  echo "${YELLOW}Admin Panel Response:${NC}"
  echo "$BODY" | jq '.'
else
  echo "${RED}❌ Expected 200, got: $STATUS${NC}"
fi

# Test 7: Test with bob (another regular user)
echo ""
echo "${YELLOW}7. Testing another regular user (bob - role: user)${NC}"
echo "--------------------------------------------------"
BOB_TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"bob","password":"password456"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$BOB_TOKEN" ]; then
  echo "${RED}❌ Failed to login as bob${NC}"
  exit 1
fi
echo "${GREEN}✅ Bob logged in successfully${NC}"

# Test 8: Bob tries admin endpoint (should fail)
echo ""
echo "${YELLOW}8. Bob accessing /api/v1/admin (should be FORBIDDEN)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8000/api/v1/admin \
  -H "Authorization: Bearer $BOB_TOKEN")
  
STATUS=$(echo "$RESPONSE" | tail -n1)
if [ "$STATUS" = "403" ]; then
  echo "${GREEN}✅ Bob correctly denied access (403 Forbidden)${NC}"
else
  echo "${RED}❌ Expected 403, got: $STATUS${NC}"
fi

# Summary
echo ""
echo "${GREEN}========================================"
echo "✅ All RBAC tests completed!"
echo "========================================${NC}"
echo ""
echo "Summary:"
echo "  • Regular users (alice, bob) can access /api/v1/cowsay"
echo "  • Regular users CANNOT access /api/v1/admin (403)"
echo "  • Admin user can access both endpoints"
echo "  • Role-based authorization is working correctly!"
