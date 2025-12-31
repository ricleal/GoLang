#!/bin/bash

# Simple quick test script
echo "🚀 Quick Test - Getting JWT and calling API"
echo ""

# Login and extract token
echo "1. Login as alice..."
TOKEN=$(curl -s -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to login"
  exit 1
fi

echo "✅ Got token!"
echo ""

# Make a request
echo "2. Calling cowsay API..."
curl -X POST http://localhost:8000/api/v1/cowsay \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Moo! This is awesome!"}' | jq '.'

echo ""
echo "✅ Done! Make multiple requests to see load balancing in action."
