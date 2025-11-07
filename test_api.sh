#!/bin/bash

# Test script for Product Service API
# Prerequisites: docker-compose up -d, and product-service running

set -e

BASE_URL="http://localhost:8080"
METRICS_URL="http://localhost:8082"

echo "=== Testing Product Service API ==="

# Test 2: Create a product
echo "2. Creating a product..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 1299.99
  }')
echo "$CREATE_RESPONSE" | jq .
PRODUCT_ID=$(echo "$CREATE_RESPONSE" | jq -r .id)
echo "Created product ID: $PRODUCT_ID"
echo

# Test 3: Create another product
echo "3. Creating another product..."
curl -s -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mouse",
    "description": "Wireless mouse",
    "price": 29.99
  }' | jq .
echo

# Test 4: List products
echo "4. Listing products..."
LIST_RESPONSE=$(curl -s "$BASE_URL/products?limit=10")
echo "$LIST_RESPONSE" | jq .
NEXT_TOKEN=$(echo "$LIST_RESPONSE" | jq -r .next_page_token)
echo "Next page token: $NEXT_TOKEN"
echo

# Test 5: Delete a product
echo "5. Deleting product $PRODUCT_ID..."
curl -s -X DELETE "$BASE_URL/products/$PRODUCT_ID" | jq .
echo

# Test 6: List products again
echo "6. Listing products again..."
curl -s "$BASE_URL/products?limit=10" | jq .
echo

# Test 7: Check metrics
echo "7. Checking Prometheus metrics..."
curl -s "$METRICS_URL/metrics" | grep -E "products_(created|deleted)_total"
echo

echo "=== All tests completed successfully ==="
