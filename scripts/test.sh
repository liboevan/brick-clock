#!/bin/bash
set -e

# Brick Clock Auth+Permission Test Script
# Usage: ./test.sh [clock_host:port] [auth_host:port]
# Default: clock=localhost:17003, auth=localhost:17001

CLOCK_API="${1:-localhost:17003}"
AUTH_API="${2:-localhost:17001}"
CLOCK_URL="http://$CLOCK_API"
AUTH_URL="http://$AUTH_API"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "\n======================================"
echo -e "Brick Clock API Auth+Permission Test"
echo -e "======================================\n"

pass() { echo -e "${GREEN}PASS${NC} $1"; }
fail() { echo -e "${RED}FAIL${NC} $1"; }
expect_code() {
  local code="$1"; shift
  local desc="$1"; shift
  local actual="$1"
  if [ "$code" = "$actual" ]; then pass "$desc"; else fail "$desc (expected $code, got $actual)"; fi
}

echo "# 1. Get admin token (admin login) ..."
ADMIN_TOKEN=$(curl -s -X POST "$AUTH_URL/login" -H "Content-Type: application/json" -d '{"username":"brick-admin","password":"brickadminpass"}' | jq -r .token)
if [ "$ADMIN_TOKEN" = "null" ] || [ -z "$ADMIN_TOKEN" ]; then fail "Admin login failed"; exit 1; fi
pass "Admin login"

echo "# 2. Get user token (user login) ..."
USER_TOKEN=$(curl -s -X POST "$AUTH_URL/login" -H "Content-Type: application/json" -d '{"username":"brick","password":"brickpass"}' | jq -r .token)
if [ "$USER_TOKEN" = "null" ] || [ -z "$USER_TOKEN" ]; then fail "User login failed"; exit 1; fi
pass "User login"

echo -e "\n# 3. Test all major endpoints with admin token (should all succeed)"
for endpoint in "/status" "/servers" "/server-mode" "/status/tracking" "/status/sources" "/status/activity" "/status/clients"; do
  echo -e "\n## GET $endpoint (admin) ..."
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $ADMIN_TOKEN" "$CLOCK_URL$endpoint")
  expect_code 200 "GET $endpoint (admin)" "$code"
done

echo -e "\n## PUT /server-mode (admin, enable) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -d '{"enabled":true}' "$CLOCK_URL/server-mode")
expect_code 200 "PUT /server-mode (admin, enable)" "$code"

echo -e "\n## PUT /servers (admin, set servers) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -d '{"servers":["pool.ntp.org","time.google.com"]}' "$CLOCK_URL/servers")
expect_code 200 "PUT /servers (admin)" "$code"

echo -e "\n## DELETE /servers (admin, reset servers) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" "$CLOCK_URL/servers")
expect_code 200 "DELETE /servers (admin)" "$code"

echo -e "\n# 4. Test endpoints with user token (should be limited by permissions)"
for endpoint in "/status" "/servers" "/server-mode" "/status/tracking" "/status/sources" "/status/activity" "/status/clients"; do
  echo -e "\n## GET $endpoint (user) ..."
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $USER_TOKEN" "$CLOCK_URL$endpoint")
  expect_code 200 "GET $endpoint (user)" "$code"
done

echo -e "\n## PUT /server-mode (user, should be forbidden) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H "Authorization: Bearer $USER_TOKEN" -H "Content-Type: application/json" -d '{"enabled":true}' "$CLOCK_URL/server-mode")
expect_code 403 "PUT /server-mode (user, forbidden)" "$code"

echo -e "\n## PUT /servers (user, should be forbidden) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT -H "Authorization: Bearer $USER_TOKEN" -H "Content-Type: application/json" -d '{"servers":["pool.ntp.org"]}' "$CLOCK_URL/servers")
expect_code 403 "PUT /servers (user, forbidden)" "$code"

echo -e "\n## DELETE /servers (user, should be forbidden) ..."
code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer $USER_TOKEN" "$CLOCK_URL/servers")
expect_code 403 "DELETE /servers (user, forbidden)" "$code"

echo -e "\nAll tests completed." 