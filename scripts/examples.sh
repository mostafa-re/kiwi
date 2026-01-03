#!/bin/bash

# kiwi API Examples
# Make sure the server is running before executing this script

BASE_URL="http://localhost:3300"

echo "======================================"
echo "kiwi API Examples"
echo "======================================"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function to print section headers
print_section() {
    echo -e "\n${BLUE}==== $1 ====${NC}\n"
}

# Helper function to print commands
print_command() {
    echo -e "${YELLOW}$ $1${NC}"
}

# Test 1: Health Check
print_section "1. Health Check"
print_command "curl $BASE_URL/health"
curl -s $BASE_URL/health | jq '.'

# Test 2: Store Simple Object
print_section "2. Store Simple Object"
print_command "curl -X PUT $BASE_URL/objects -d '{\"key\":\"greeting\",\"value\":\"Hello World\"}'"
curl -s -X PUT $BASE_URL/objects \
  -H "Content-Type: application/json" \
  -d '{
    "key": "greeting",
    "value": "Hello World"
  }' | jq '.'

# Test 3: Retrieve Object
print_section "3. Retrieve Object"
print_command "curl $BASE_URL/objects/greeting"
curl -s $BASE_URL/objects/greeting | jq '.'

# Test 4: Store Complex Object (User)
print_section "4. Store Complex User Object"
print_command "curl -X PUT '$BASE_URL/objects?collection=users' -d '{...}'"
curl -s -X PUT "$BASE_URL/objects?collection=users" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "john_doe",
    "value": {
      "id": "user_001",
      "name": "John Doe",
      "email": "john.doe@example.com",
      "age": 30,
      "role": "admin",
      "metadata": {
        "created_at": "2025-10-31T10:00:00Z",
        "last_login": "2025-10-31T12:00:00Z"
      }
    }
  }' | jq '.'

# Test 5: Store Another User
print_section "5. Store Another User"
curl -s -X PUT "$BASE_URL/objects?collection=users" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "jane_smith",
    "value": {
      "id": "user_002",
      "name": "Jane Smith",
      "email": "jane.smith@example.com",
      "age": 28,
      "role": "developer"
    }
  }' | jq '.'

# Test 6: Retrieve User
print_section "6. Retrieve User from Collection"
print_command "curl '$BASE_URL/objects/john_doe?collection=users'"
curl -s "$BASE_URL/objects/john_doe?collection=users" | jq '.'

# Test 7: Store Product Objects
print_section "7. Store Product Objects"
print_command "curl -X PUT '$BASE_URL/objects?collection=products' -d '{...}'"

# Product 1
curl -s -X PUT "$BASE_URL/objects?collection=products" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "laptop_001",
    "value": {
      "id": "prod_001",
      "name": "MacBook Pro 16\"",
      "category": "Laptops",
      "price": 2499.99,
      "stock": 15,
      "specs": {
        "cpu": "M3 Max",
        "ram": "32GB",
        "storage": "1TB SSD"
      }
    }
  }' | jq '.'

# Product 2
curl -s -X PUT "$BASE_URL/objects?collection=products" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "mouse_001",
    "value": {
      "id": "prod_002",
      "name": "Logitech MX Master 3",
      "category": "Accessories",
      "price": 99.99,
      "stock": 50
    }
  }' | jq '.'

# Test 8: List All Objects in Default Collection
print_section "8. List All Objects (Default Collection)"
print_command "curl $BASE_URL/objects"
curl -s $BASE_URL/objects | jq '.'

# Test 9: List All Users
print_section "9. List All Users"
print_command "curl '$BASE_URL/objects?collection=users'"
curl -s "$BASE_URL/objects?collection=users" | jq '.'

# Test 10: List All Products
print_section "10. List All Products"
print_command "curl '$BASE_URL/objects?collection=products'"
curl -s "$BASE_URL/objects?collection=products" | jq '.'

# Test 11: Store Order Object with Array
print_section "11. Store Complex Order Object"
curl -s -X PUT "$BASE_URL/objects?collection=orders" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "order_12345",
    "value": {
      "order_id": "ORD-12345",
      "customer_id": "user_001",
      "items": [
        {
          "product_id": "prod_001",
          "product_name": "MacBook Pro 16\"",
          "quantity": 1,
          "price": 2499.99
        },
        {
          "product_id": "prod_002",
          "product_name": "Logitech MX Master 3",
          "quantity": 2,
          "price": 99.99
        }
      ],
      "total": 2699.97,
      "status": "pending",
      "shipping_address": {
        "street": "123 Main St",
        "city": "San Francisco",
        "state": "CA",
        "zip": "94105"
      },
      "created_at": "2025-10-31T14:30:00Z"
    }
  }' | jq '.'

# Test 12: Retrieve Order
print_section "12. Retrieve Order"
print_command "curl '$BASE_URL/objects/order_12345?collection=orders'"
curl -s "$BASE_URL/objects/order_12345?collection=orders" | jq '.'

# Test 13: Update Object (by storing with same key)
print_section "13. Update Object"
print_command "curl -X PUT '$BASE_URL/objects?collection=products' -d '{...}'"
curl -s -X PUT "$BASE_URL/objects?collection=products" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "laptop_001",
    "value": {
      "id": "prod_001",
      "name": "MacBook Pro 16\"",
      "category": "Laptops",
      "price": 2299.99,
      "stock": 12,
      "specs": {
        "cpu": "M3 Max",
        "ram": "32GB",
        "storage": "1TB SSD"
      },
      "on_sale": true
    }
  }' | jq '.'

# Test 14: Verify Update
print_section "14. Verify Update"
curl -s "$BASE_URL/objects/laptop_001?collection=products" | jq '.'

# Test 15: Try to Get Non-existent Key (404)
print_section "15. Try to Get Non-existent Key (Should return 404)"
print_command "curl '$BASE_URL/objects/nonexistent?collection=users'"
curl -s -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/objects/nonexistent?collection=users" | jq '.' 2>/dev/null || echo "Key not found"

# Test 16: Delete Object
print_section "16. Delete Object"
print_command "curl -X DELETE '$BASE_URL/objects/mouse_001?collection=products'"
curl -s -X DELETE "$BASE_URL/objects/mouse_001?collection=products" | jq '.'

# Test 17: Verify Deletion
print_section "17. Verify Deletion (Should return 404)"
curl -s -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/objects/mouse_001?collection=products" | jq '.' 2>/dev/null || echo "Key deleted successfully"

# Test 18: List Products After Deletion
print_section "18. List Products After Deletion"
curl -s "$BASE_URL/objects?collection=products" | jq '.'

# Test 19: Store JSON Array as Value
print_section "19. Store JSON Array as Value"
curl -s -X PUT "$BASE_URL/objects?collection=settings" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "allowed_countries",
    "value": ["US", "CA", "UK", "DE", "FR", "JP"]
  }' | jq '.'

# Test 20: Store Number as Value
print_section "20. Store Number as Value"
curl -s -X PUT "$BASE_URL/objects?collection=counters" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "page_views",
    "value": 12345
  }' | jq '.'

# Summary
print_section "Summary"
echo -e "${GREEN}All examples completed!${NC}"
echo ""
echo "Collections created:"
echo "  - default (greeting)"
echo "  - users (john_doe, jane_smith)"
echo "  - products (laptop_001)"
echo "  - orders (order_12345)"
echo "  - settings (allowed_countries)"
echo "  - counters (page_views)"
echo ""
echo -e "${GREEN}Server is running at: $BASE_URL${NC}"
