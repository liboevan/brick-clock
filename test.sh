#!/bin/bash

# Test script for el/chrony-suite API endpoints
# Usage: ./test_api.sh [host:port]
# Default: localhost:8291

API_BASE="${1:-localhost:8291}"
BASE_URL="http://$API_BASE"

echo "Testing el/chrony-suite API at $BASE_URL"
echo "======================================"

# Function to test an endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "\n--- Testing: $description ---"
    echo "Endpoint: $method $endpoint"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$BASE_URL$endpoint")
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X PUT -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint")
    elif [ "$method" = "DELETE" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X DELETE "$BASE_URL$endpoint")
    fi
    
    # Extract status code
    status_code=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    # Extract response body
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')
    
    echo "Status: $status_code"
    echo "Response:"
    echo "$body" | jq . 2>/dev/null || echo "$body"
}

# Test all endpoints
echo "Starting API tests..."

# 1. Get chrony version
test_endpoint "GET" "/chrony/version" "" "Get chrony version"

# 2. Get current status
test_endpoint "GET" "/chrony/status" "" "Get current status"

# 3. Get current sources
test_endpoint "GET" "/chrony/servers" "" "Get current sources"

# 4. Get server mode status
test_endpoint "GET" "/chrony/server-mode" "" "Get server mode status"

# 5. Test setting servers
test_endpoint "PUT" "/chrony/servers" '{"servers": ["pool.ntp.org", "time.google.com"]}' "Set NTP servers"

# 6. Get sources again to verify change
test_endpoint "GET" "/chrony/servers" "" "Get sources after setting servers"

# 7. Test setting default servers
test_endpoint "PUT" "/chrony/servers/default" "" "Set default servers"

# 8. Get sources again to verify default
test_endpoint "GET" "/chrony/servers" "" "Get sources after setting defaults"

# 9. Test resetting servers
test_endpoint "DELETE" "/chrony/servers" "" "Reset servers"

# 10. Get sources again to verify reset
test_endpoint "GET" "/chrony/servers" "" "Get sources after reset"

# 11. Test setting server mode
test_endpoint "PUT" "/chrony/server-mode" '{"enabled": true}' "Enable server mode"

# 12. Get server mode to verify
test_endpoint "GET" "/chrony/server-mode" "" "Get server mode after enabling"

# 13. Test disabling server mode
test_endpoint "PUT" "/chrony/server-mode" '{"enabled": false}' "Disable server mode"

# 14. Get server mode to verify
test_endpoint "GET" "/chrony/server-mode" "" "Get server mode after disabling"

# 15. Final status check
test_endpoint "GET" "/chrony/status" "" "Final status check"

echo -e "\n======================================"
echo "API testing completed!"
echo "Check the responses above for any errors." 