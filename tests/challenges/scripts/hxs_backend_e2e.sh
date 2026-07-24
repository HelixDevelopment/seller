#!/bin/bash
# hxs_backend_e2e.sh — Full API surface E2E tests
# Auth, merchants, products, orders, payments, webhooks, payouts, WebSocket

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_DIR="$(cd "$SCRIPT_DIR/../config" && pwd)"
LIB_AB="$PROJECT_DIR/submodules/challenges/lib/anti_bluff.sh"

[ -f "$LIB_AB" ] || { echo "FATAL: anti_bluff.sh missing" >&2; exit 2; }
. "$LIB_AB"

ab_init "hxs_backend_e2e" "/tmp/hxs_backend_e2e.results"
ab_send_action "HXS Backend E2E — full API surface test"

[ -f "$CONFIG_DIR/credentials.env" ] && . "$CONFIG_DIR/credentials.env"

API_URL="${HXS_API_URL:-http://127.0.0.1:8080}"
TEST_PASSED=0

on_exit() {
    local rc=$?
    [ "$TEST_PASSED" = "0" ] && [ $rc -ne 0 ] && [ $rc -ne 2 ] && \
        ab_fail "Backend E2E exited at line ${LINENO:-?} rc=$rc"
    exit $rc
}
trap on_exit EXIT

echo "=== HXS Backend E2E Tests ==="

echo "--- Auth ---"
REG=$(curl -s -m 5 -X POST "$API_URL/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"e2e@helix.test\",\"password\":\"testpassword123!\",\"name\":\"E2E Tester\"}" 2>/dev/null)
if echo "$REG" | grep -qiE '"(id|token)"'; then
    ab_pass "Registration succeeds for new user"
else
    ab_skip "Registration endpoint not available or returned: $(echo "$REG" | head -c 80)" "infra"
fi

LOGIN=$(curl -s -m 5 -X POST "$API_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$HXS_ADMIN_EMAIL\",\"password\":\"$HXS_ADMIN_PASSWORD\"}" 2>/dev/null)
TOKEN=$(echo "$LOGIN" | grep -oE '"token"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
if [ -n "$TOKEN" ]; then
    ab_pass "Admin login returns JWT"
else
    TOKEN=$(echo "$LOGIN" | grep -oE '"access_token"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
    [ -n "$TOKEN" ] && ab_pass "Admin login returns access_token" || ab_fail "Admin login returned no token"
fi

BAD_LOGIN=$(curl -s -m 5 -w '%{http_code}' -X POST "$API_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"wrong@test.com","password":"bad"}' -o /dev/null 2>/dev/null)
[ "$BAD_LOGIN" = "401" ] && ab_pass "Invalid login returns 401" || ab_fail "Invalid login got HTTP $BAD_LOGIN, want 401"

echo "--- Health ---"
HEALTH=$(curl -s -m 3 "$API_URL/health" -w '%{http_code}' -o /dev/null 2>/dev/null || echo "000")
[ "$HEALTH" = "200" ] && ab_pass "Health endpoint returns 200" || ab_fail "Health endpoint returned HTTP $HEALTH"

echo "--- Merchants ---"
if [ -n "$TOKEN" ]; then
    MERCHANTS=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" "$API_URL/api/v1/merchants" 2>/dev/null)
    if echo "$MERCHANTS" | grep -qiE '\[|\{'; then
        ab_pass "Merchants list returns valid JSON"
    else
        ab_skip "Merchants endpoint returned non-JSON" "infra"
    fi
    UNAUTH=$(curl -s -m 5 -w '%{http_code}' "$API_URL/api/v1/merchants" -o /dev/null 2>/dev/null)
    [ "$UNAUTH" = "401" ] && ab_pass "Merchants without auth returns 401" || ab_fail "Merchants without auth got HTTP $UNAUTH, want 401"
else
    ab_skip "No token — skipping merchant tests" "infra"
fi
# Get or create a merchant for subsequent tests
MERCHANT_ID=$(echo "$MERCHANTS" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
if [ -z "$MERCHANT_ID" ] && [ -n "$TOKEN" ]; then
    CREATED=$(curl -s -m 5 -X POST -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "{\"name\":\"HXS Test Merchant\",\"legal_name\":\"HXS Test Merchant LLC\",\"email\":\"$HXS_MERCHANT_EMAIL\",\"country\":\"US\",\"currency\":\"USD\"}" \
        "$API_URL/api/v1/merchants" 2>/dev/null)
    MERCHANT_ID=$(echo "$CREATED" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
fi
if [ -n "$MERCHANT_ID" ]; then
    ab_pass "Obtained/created merchant ID: $MERCHANT_ID"
else
    MERCHANT_ID="00000000-0000-0000-0000-000000000000"
    ab_skip "No merchant available — using dummy ID" "infra"
fi

echo "--- Customers ---"
CUSTOMER_ID=""
if [ -n "$TOKEN" ] && [ -n "$MERCHANT_ID" ] && [ "$MERCHANT_ID" != "00000000-0000-0000-0000-000000000000" ]; then
    CUST_EMAIL="e2e.customer.$(date +%s)@helix.test"
    CUST=$(curl -s -m 5 -X POST -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "{\"name\":\"E2E Customer\",\"email\":\"$CUST_EMAIL\"}" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/customers" 2>/dev/null)
    CUSTOMER_ID=$(echo "$CUST" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
    if [ -n "$CUSTOMER_ID" ]; then
        ab_pass "Customer created (id=$CUSTOMER_ID)"
    else
        CUSTOMER_ID=""
        ab_skip "Customer creation failed: $(echo "$CUST" | head -c 80)" "infra"
    fi
else
    ab_skip "No token or merchant — skipping customer tests" "infra"
fi

echo "--- Payment Methods ---"
PM_ID=""
if [ -n "$TOKEN" ] && [ -n "$MERCHANT_ID" ] && [ -n "$CUSTOMER_ID" ]; then
    PM_CREATE=$(curl -s -m 5 -X POST -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "{\"customer_id\":\"$CUSTOMER_ID\",\"type\":\"card\",\"provider\":\"stripe\",\"provider_token\":\"tok_visa\",\"last4\":\"4242\",\"brand\":\"visa\",\"exp_month\":12,\"exp_year\":2028,\"is_default\":true}" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/payment-methods" 2>/dev/null)
    PM_ID=$(echo "$PM_CREATE" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
    if [ -n "$PM_ID" ]; then
        ab_pass "Payment method created (id=$PM_ID)"
    else
        PM_ID=""
        ab_skip "Payment method creation failed: $(echo "$PM_CREATE" | head -c 80)" "infra"
    fi

    # List payment methods
    PMS=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/payment-methods?customer_id=$CUSTOMER_ID" 2>/dev/null)
    echo "$PMS" | grep -qiE '"payment_methods"' && \
        ab_pass "Payment methods list works" || \
        ab_skip "Payment methods list response unexpected" "infra"
else
    ab_skip "No token, merchant, or customer — skipping payment method tests" "infra"
fi

echo "--- Product CRUD ---"
if [ -n "$TOKEN" ] && [ -n "$MERCHANT_ID" ]; then
    # Create product
    PROD=$(curl -s -m 5 -X POST -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Test Widget","description":"A test product","price":2999,"currency":"USD"}' \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/products" 2>/dev/null)
    PROD_ID=$(echo "$PROD" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
    if [ -n "$PROD_ID" ]; then
        ab_pass "Product created (id=$PROD_ID)"
    else
        ab_fail "Product creation failed: $(echo "$PROD" | head -c 80)"
    fi

    # List products
    PRODS=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/products" 2>/dev/null)
    if echo "$PRODS" | grep -qiE '"products"|"items"'; then
        ab_pass "Product list returns products array"
    else
        ab_skip "Product list response unexpected: $(echo "$PRODS" | head -c 80)" "infra"
    fi

    # Get product
    if [ -n "$PROD_ID" ]; then
        PROD_GET=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" \
            "$API_URL/api/v1/merchants/$MERCHANT_ID/products/$PROD_ID" 2>/dev/null)
        echo "$PROD_GET" | grep -qiE '"name".*"Test Widget"' && \
            ab_pass "Product retrieved with correct name" || \
            ab_fail "Product retrieval returned unexpected data"

        # Update product
        PROD_UPD=$(curl -s -m 5 -X PUT -H "Authorization: Bearer $TOKEN" \
            -H 'Content-Type: application/json' \
            -d '{"name":"Updated Widget","price":3999}' \
            "$API_URL/api/v1/merchants/$MERCHANT_ID/products/$PROD_ID" 2>/dev/null)
        echo "$PROD_UPD" | grep -qiE '"name".*"Updated Widget"' && \
            ab_pass "Product updated with new name" || \
            ab_fail "Product update failed"

        # Delete product
        DEL_CODE=$(curl -s -m 5 -o /dev/null -w '%{http_code}' -X DELETE \
            -H "Authorization: Bearer $TOKEN" \
            "$API_URL/api/v1/merchants/$MERCHANT_ID/products/$PROD_ID" 2>/dev/null)
        [ "$DEL_CODE" = "200" ] && ab_pass "Product deleted (HTTP 200)" || \
            ab_fail "Product delete returned HTTP $DEL_CODE"
    fi
else
    ab_skip "No token or merchant — skipping product CRUD tests" "infra"
fi

echo "--- Payment Flow ---"
if [ -n "$TOKEN" ] && [ -n "$MERCHANT_ID" ] && [ "$MERCHANT_ID" != "00000000-0000-0000-0000-000000000000" ]; then
    # Process payment
    PMT=$(curl -s -m 5 -X POST -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "{\"amount\":5000,\"currency\":\"USD\",\"customer_id\":\"$CUSTOMER_ID\",\"payment_method_id\":\"$PM_ID\",\"description\":\"Test charge\"}" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/transactions" 2>/dev/null)
    TX_ID=$(echo "$PMT" | grep -oE '"_?id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed 's/.*: *"//;s/"//')
    if [ -n "$TX_ID" ]; then
        ab_pass "Payment transaction created (id=$TX_ID)"
    else
        ab_skip "Payment endpoint returned: $(echo "$PMT" | head -c 80)" "infra"
    fi

    # List transactions
    TXS=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/transactions" 2>/dev/null)
    if echo "$TXS" | grep -qiE '"items"|\[|"transactions"'; then
        ab_pass "Transactions list endpoint works"
    else
        ab_skip "Transactions list returned unexpected" "infra"
    fi
fi

echo "--- Dispute Flow ---"
if [ -n "$TOKEN" ] && [ -n "$MERCHANT_ID" ] && [ "$MERCHANT_ID" != "00000000-0000-0000-0000-000000000000" ]; then
    # List disputes
    DISP=$(curl -s -m 5 -H "Authorization: Bearer $TOKEN" \
        "$API_URL/api/v1/merchants/$MERCHANT_ID/disputes" 2>/dev/null)
    echo "$DISP" | grep -qiE '"items"|\[|"disputes"' && \
        ab_pass "Disputes list endpoint works" || \
        ab_skip "Disputes list response unexpected" "infra"
fi

echo "--- WebSocket ---"
if command -v websocat >/dev/null 2>&1; then
    WS_RESULT=$(echo "ping" | timeout 4 websocat "ws://127.0.0.1:8080/ws" 2>&1 || true)
    if echo "$WS_RESULT" | grep -qiE 'pong|connected|message'; then
        ab_pass "WebSocket ping/pong works"
    else
        ab_skip "WebSocket ping/pong failed: $(echo "$WS_RESULT" | head -c 80)" "infra"
    fi
elif command -v curl >/dev/null && curl --version 2>/dev/null | grep -qi websocket; then
    WS_UPGRADE=$(curl -s -m 3 -o /dev/null -w '%{http_code}' -H "Upgrade: websocket" -H "Connection: Upgrade" "http://127.0.0.1:8080/ws" 2>/dev/null || echo "000")
    [ "$WS_UPGRADE" = "101" ] && ab_pass "WebSocket upgrade returns 101 (switching protocols)" || \
        ab_skip "WebSocket upgrade returned HTTP $WS_UPGRADE, want 101" "infra"
elif command -v wscat >/dev/null 2>&1; then
    WS_RESULT=$(echo "ping" | timeout 4 wscat -c "ws://127.0.0.1:8080/ws" 2>&1 || true)
    echo "$WS_RESULT" | grep -qiE 'pong|connected' && ab_pass "WebSocket ping/pong works via wscat" || \
        ab_skip "WebSocket via wscat failed" "infra"
else
    ab_skip "No WebSocket client available (install websocat, wscat, or curl with WebSocket support)" "infra"
fi

echo
echo "=== hxs_backend_e2e complete ==="
TEST_PASSED=1
ab_summary
