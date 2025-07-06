#!/bin/bash
set -e

# Source shared configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

print_header "API Testing"

# Test script for Brick Clock API endpoints
# Usage: ./test.sh [host:port]
# Default: localhost:17003

API_BASE="${1:-localhost:$API_PORT}"
BASE_URL="http://$API_BASE"

print_info "Testing Brick Clock API at $BASE_URL"
echo "======================================"

# Function to test an endpoint with proper error handling
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "\n--- Testing: $description ---"
    echo "Endpoint: $method $endpoint"
    
    local response
    local status_code
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$BASE_URL$endpoint" 2>/dev/null || echo "HTTP_STATUS:000")
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X PUT -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint" 2>/dev/null || echo "HTTP_STATUS:000")
    elif [ "$method" = "DELETE" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X DELETE "$BASE_URL$endpoint" 2>/dev/null || echo "HTTP_STATUS:000")
    fi
    
    # Extract status code
    status_code=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    # Extract response body
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')
    
    # Color code the status
    if [ "$status_code" -ge 200 ] && [ "$status_code" -lt 300 ]; then
        echo -e "Status: ${GREEN}$status_code${NC}"
    elif [ "$status_code" -ge 400 ] && [ "$status_code" -lt 500 ]; then
        echo -e "Status: ${YELLOW}$status_code${NC}"
    else
        echo -e "Status: ${RED}$status_code${NC}"
    fi
    
    echo "Response:"
    if command -v jq >/dev/null 2>&1; then
        echo "$body" | jq . 2>/dev/null || echo "$body"
    else
        echo "$body"
    fi
    
    # Return status for summary
    if [ "$status_code" -ge 200 ] && [ "$status_code" -lt 300 ]; then
        return 0
    else
        return 1
    fi
}

# Initialize test counters
total_tests=0
passed_tests=0
failed_tests=0

# Function to run a test and track results
run_test() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    total_tests=$((total_tests + 1))
    
    if test_endpoint "$method" "$endpoint" "$data" "$description"; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
}

print_info "Starting API tests..."

# Test all endpoints
echo ""

# 1. Health check
run_test "GET" "/health" "" "Health check"

# 2. Get application version
run_test "GET" "/version" "" "Get application version"

# 3. Get app version (alternative endpoint)
run_test "GET" "/app-version" "" "Get app version"

# 4. Get current status
run_test "GET" "/status" "" "Get current status"

# 5. Get status with specific flags
run_test "GET" "/status?flags=23" "" "Get status with tracking + sources + activity + server mode"

# 6. Get tracking information
run_test "GET" "/status/tracking" "" "Get tracking information"

# 7. Get sources information
run_test "GET" "/status/sources" "" "Get sources information"

# 8. Get activity information
run_test "GET" "/status/activity" "" "Get activity information"

# 9. Get clients information
run_test "GET" "/status/clients" "" "Get clients information"

# 10. Get current servers
run_test "GET" "/servers" "" "Get current servers"

# 11. Test setting servers
run_test "PUT" "/servers" '{"servers": ["pool.ntp.org", "time.google.com"]}' "Set NTP servers"

# 12. Get servers again to verify change
run_test "GET" "/servers" "" "Get servers after setting custom servers"

# 13. Test setting default servers
run_test "PUT" "/servers/default" "" "Set default servers"

# 14. Get servers again to verify default
run_test "GET" "/servers" "" "Get servers after setting defaults"

# 15. Test resetting servers
run_test "DELETE" "/servers" "" "Reset servers"

# 16. Get servers again to verify reset
run_test "GET" "/servers" "" "Get servers after reset"

# 17. Get server mode status
run_test "GET" "/server-mode" "" "Get server mode status"

# 18. Test enabling server mode
run_test "PUT" "/server-mode" '{"enabled": true}' "Enable server mode"

# 19. Get server mode to verify
run_test "GET" "/server-mode" "" "Get server mode after enabling"

# 20. Test disabling server mode
run_test "PUT" "/server-mode" '{"enabled": false}' "Disable server mode"

# 21. Get server mode to verify
run_test "GET" "/server-mode" "" "Get server mode after disabling"

# 22. Final status check
run_test "GET" "/status" "" "Final status check"

# Print test summary
echo -e "\n======================================"
echo -e "${BLUE}Test Summary:${NC}"
echo -e "Total Tests: $total_tests"
echo -e "Passed: ${GREEN}$passed_tests${NC}"
echo -e "Failed: ${RED}$failed_tests${NC}"

if [ $failed_tests -eq 0 ]; then
    print_info "All tests passed! 🎉"
    exit 0
else
    print_error "Some tests failed. Please check the responses above."
    exit 1
fi 