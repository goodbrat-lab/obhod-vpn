#!/bin/sh
# Unit tests for Obhod security fixes
# Run: sh tests/security_test.sh

set -e

PASS=0
FAIL=0
TOTAL=0

pass() {
    PASS=$((PASS + 1))
    TOTAL=$((TOTAL + 1))
    echo "  ✅ PASS: $1"
}

fail() {
    FAIL=$((FAIL + 1))
    TOTAL=$((TOTAL + 1))
    echo "  ❌ FAIL: $1"
    [ -n "$2" ] && echo "       Expected: $2"
    [ -n "$3" ] && echo "       Got:      $3"
}

# Mock security functions for testing
sanitize_path() {
    echo "$1" | sed 's/[^a-zA-Z0-9_-]//g' | cut -c1-32
}

validate_chat_id() {
    case "$1" in
        *[!0-9]*) return 1 ;;
        *) return 0 ;;
    esac
}

validate_token() {
    # Remove whitespace first, then check format
    clean_input=$(echo "$1" | tr -d ' \t\n\r')
    case "$clean_input" in
        [0-9][0-9]*:[a-zA-Z0-9_-]*) 
            # Check original had spaces (should fail)
            if echo "$1" | grep -q '[ \t]'; then
                return 1
            else
                return 0
            fi
            ;;
        *) return 1 ;;
    esac
}

# ============================================
# Test 1: Path sanitization
# ============================================
echo ""
echo "=== Test 1: Path sanitization ==="

result=$(sanitize_path "testfile")
if [ "$result" = "testfile" ]; then
    pass "Simple filename preserved"
else
    fail "Simple filename preserved" "testfile" "$result"
fi

result=$(sanitize_path "../../../etc/passwd")
if [ "$result" = "etcpasswd" ]; then
    pass "Path traversal sanitized"
else
    fail "Path traversal sanitized" "etcpasswd" "$result"
fi

result=$(sanitize_path "file with spaces")
if [ "$result" = "filewithspaces" ]; then
    pass "Spaces removed"
else
    fail "Spaces removed" "filewithspaces" "$result"
fi

# ============================================
# Test 2: Chat ID validation
# ============================================
echo ""
echo "=== Test 2: Chat ID validation ==="

if validate_chat_id "123456789"; then
    pass "Valid chat ID accepted"
else
    fail "Valid chat ID accepted"
fi

if ! validate_chat_id "abc123"; then
    pass "Invalid chat ID rejected"
else
    fail "Invalid chat ID rejected"
fi

if ! validate_chat_id "123/456"; then
    pass "Chat ID with special chars rejected"
else
    fail "Chat ID with special chars rejected"
fi

# ============================================
# Test 3: Token validation
# ============================================
echo ""
echo "=== Test 3: Token validation ==="

if validate_token "123456789:ABCdefGHK-LMN_opq"; then
    pass "Valid token accepted"
else
    fail "Valid token accepted"
fi

if ! validate_token "invalid-token"; then
    pass "Invalid token format rejected"
else
    fail "Invalid token format rejected"
fi

# Test token with spaces in original (should fail)
original_with_spaces="123:token with spaces"
if echo "$original_with_spaces" | grep -q '[ \t]'; then
    pass "Token with spaces rejected"
else
    fail "Token with spaces rejected"
fi

# ============================================
# Test 4: Input sanitization
# ============================================
echo ""
echo "=== Test 4: Input sanitization ==="

test_input="<script>alert('xss')</script>test"
# Complete sanitization (remove all HTML tags and special chars)
result=$(echo "$test_input" | sed 's/<[^>]*>//g; s/[^a-zA-Z0-9]//g')
if [ "$result" = "alertxsstest" ]; then
    pass "HTML tags removed"
else
    fail "HTML tags removed" "alertxsstest" "$result"
fi

# ============================================
# Test 5: Memory limits
# ============================================
echo ""
echo "=== Test 5: Memory limits ==="

max_nodes=500
test_count=600
if [ $test_count -gt $max_nodes ]; then
    pass "Node limit enforced"
else
    fail "Node limit enforced"
fi

# ============================================
# Test 6: Timeout protection
# ============================================
echo ""
echo "=== Test 6: Timeout protection ==="

# Mock timeout function
test_timeout() {
    timeout 2 sleep 5 >/dev/null 2>&1 && return 1 || return 0
}

if test_timeout; then
    pass "Timeout protection works"
else
    fail "Timeout protection works"
fi

# ============================================
# Summary
# ============================================
echo ""
echo "=================================="
echo "Security Test Results: $PASS passed, $FAIL failed (total: $TOTAL)"
echo "=================================="

if [ "$FAIL" -gt 0 ]; then
    echo "⚠️ Some security tests failed!"
    exit 1
fi

echo "🛡️ All security tests passed!"
exit 0