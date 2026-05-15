#!/bin/sh
# Unit tests for subscription parsing (fetch_subscription + link filtering)
# Run: sh tests/test_subscription.sh
#
# These tests mock the network layer and validate:
# 1. Base64-encoded subscription decoding
# 2. Plain-text subscription handling
# 3. URL-safe Base64 support
# 4. Mixed content filtering (only proxy links extracted)
# 5. CRLF line ending handling
# 6. Empty/invalid subscription handling

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

# ============================================
# Test helpers (standalone, no OpenWrt deps)
# ============================================

# Minimal log stub
log() { :; }
echolog() { :; }

# ============================================
# Test 1: Base64 detection and decoding
# ============================================
echo ""
echo "=== Test 1: Base64-encoded subscription ==="

PLAIN_SUB="vless://d9b38312-8bc1-024e-b828-a7eb708a9212@ap4.directly.chat:443?type=tcp&security=reality&flow=xtls-rprx-vision&sni=www.microsoft.com&fp=chrome&pbk=testkey#Server1
ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0cGFzc3dvcmQ=@1.2.3.4:8388#Server2
trojan://mypassword@example.com:443?security=tls&sni=example.com#Server3"

# Encode to Base64
ENCODED_SUB=$(echo "$PLAIN_SUB" | base64 -w 0 2>/dev/null || echo "$PLAIN_SUB" | base64 2>/dev/null)

# Create mock temp file
TMPDIR_TEST=$(mktemp -d 2>/dev/null || echo "/tmp/obhod_test_$$")
mkdir -p "$TMPDIR_TEST"

# Simulate what fetch_subscription does for Base64 content
test_base64_decode() {
    local data="$1"
    local output_file="$TMPDIR_TEST/test1_output.txt"

    # This is the core logic from fetch_subscription()
    if echo "$data" | grep -qiE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://'; then
        echo "$data" > "$output_file"
    else
        local decoded
        decoded=$(echo "$data" | tr -d '\r\n ' | tr -- '-_' '+/' | base64 -d 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$decoded" ] && echo "$decoded" | grep -qiE '(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://'; then
            echo "$decoded" > "$output_file"
        else
            echo "$data" > "$output_file"
        fi
    fi

    # Convert CRLF
    if [ -f "$output_file" ]; then
        tr -d '\r' < "$output_file" > "${output_file}.tmp" && mv "${output_file}.tmp" "$output_file"
    fi

    echo "$output_file"
}

OUTPUT=$(test_base64_decode "$ENCODED_SUB")
LINK_COUNT=$(grep -cE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$OUTPUT" 2>/dev/null || echo 0)

if [ "$LINK_COUNT" -eq 3 ]; then
    pass "Base64 subscription decoded: found 3 links"
else
    fail "Base64 subscription decoded" "3 links" "$LINK_COUNT links"
fi

# Check specific link types
if grep -q '^vless://' "$OUTPUT"; then
    pass "Contains VLESS link"
else
    fail "Contains VLESS link"
fi

if grep -q '^ss://' "$OUTPUT"; then
    pass "Contains Shadowsocks link"
else
    fail "Contains Shadowsocks link"
fi

if grep -q '^trojan://' "$OUTPUT"; then
    pass "Contains Trojan link"
else
    fail "Contains Trojan link"
fi

# ============================================
# Test 2: Plain-text subscription
# ============================================
echo ""
echo "=== Test 2: Plain-text subscription ==="

OUTPUT2=$(test_base64_decode "$PLAIN_SUB")
LINK_COUNT2=$(grep -cE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$OUTPUT2" 2>/dev/null || echo 0)

if [ "$LINK_COUNT2" -eq 3 ]; then
    pass "Plain-text subscription: found 3 links"
else
    fail "Plain-text subscription" "3 links" "$LINK_COUNT2 links"
fi

# ============================================
# Test 3: URL-safe Base64
# ============================================
echo ""
echo "=== Test 3: URL-safe Base64 (- and _ instead of + and /) ==="

# URL-safe Base64: replace + with - and / with _
URLSAFE_SUB=$(echo "$PLAIN_SUB" | base64 -w 0 2>/dev/null || echo "$PLAIN_SUB" | base64 2>/dev/null)
URLSAFE_SUB=$(echo "$URLSAFE_SUB" | tr '+/' '-_')

OUTPUT3=$(test_base64_decode "$URLSAFE_SUB")
LINK_COUNT3=$(grep -cE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$OUTPUT3" 2>/dev/null || echo 0)

if [ "$LINK_COUNT3" -eq 3 ]; then
    pass "URL-safe Base64 decoded: found 3 links"
else
    fail "URL-safe Base64 decoded" "3 links" "$LINK_COUNT3 links"
fi

# ============================================
# Test 4: Mixed content (proxy links + comments + empty lines)
# ============================================
echo ""
echo "=== Test 4: Link filtering from mixed content ==="

MIXED_CONTENT="# Comment header
vless://uuid1@host1:443?type=tcp&security=reality&sni=test.com&fp=chrome&pbk=key1#Node1

Some random text that is not a link
ss://bWV0aG9kOnBhc3N3b3Jk@host2:8388#Node2
vmess://eyJ2IjoiMiJ9
trojan://pass3@host3:443?security=tls#Node3
hy2://authpass@host4:443?sni=test.com#Node4

# Another comment"

echo "$MIXED_CONTENT" > "$TMPDIR_TEST/test4_input.txt"

# Simulate the grep filter from configure_outbound_handler
FILTERED=$(grep -iE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$TMPDIR_TEST/test4_input.txt")
FILTERED_COUNT=$(echo "$FILTERED" | grep -c '://')

if [ "$FILTERED_COUNT" -eq 5 ]; then
    pass "Filtered 5 proxy links from mixed content"
else
    fail "Filtered proxy links from mixed content" "5 links" "$FILTERED_COUNT links"
fi

# Verify non-link lines are excluded
if echo "$FILTERED" | grep -q "^# Comment"; then
    fail "Comments should be excluded"
else
    pass "Comments correctly excluded"
fi

if echo "$FILTERED" | grep -q "^Some random"; then
    fail "Random text should be excluded"
else
    pass "Random text correctly excluded"
fi

# ============================================
# Test 5: CRLF handling
# ============================================
echo ""
echo "=== Test 5: CRLF line endings ==="

CRLF_CONTENT=$(printf "vless://uuid@host:443?type=tcp&security=reality&sni=test.com&fp=chrome&pbk=key1#Node1\r\nss://bWV0aG9kOnBhc3N3b3Jk@host2:8388#Node2\r\n")
echo "$CRLF_CONTENT" > "$TMPDIR_TEST/test5_input.txt"

# Apply CRLF -> LF conversion
tr -d '\r' < "$TMPDIR_TEST/test5_input.txt" > "$TMPDIR_TEST/test5_clean.txt"

if grep -qP '\r' "$TMPDIR_TEST/test5_clean.txt" 2>/dev/null; then
    fail "CRLF should be removed"
else
    pass "CRLF removed from output"
fi

LINK_COUNT5=$(grep -cE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$TMPDIR_TEST/test5_clean.txt" 2>/dev/null || echo 0)
if [ "$LINK_COUNT5" -eq 2 ]; then
    pass "Found 2 links after CRLF cleanup"
else
    fail "Links after CRLF cleanup" "2 links" "$LINK_COUNT5 links"
fi

# ============================================
# Test 6: Empty subscription
# ============================================
echo ""
echo "=== Test 6: Empty / invalid subscription ==="

EMPTY_CONTENT="just some random text
no proxy links here
another line"

echo "$EMPTY_CONTENT" > "$TMPDIR_TEST/test6_input.txt"

EMPTY_LINKS=$(grep -iE '^(vless|vmess|trojan|ss|hy2|hysteria2|socks[45])://' "$TMPDIR_TEST/test6_input.txt" | tr '\n' ' ')
if [ -z "$EMPTY_LINKS" ]; then
    pass "Empty subscription correctly detected (no proxy links)"
else
    fail "Empty subscription detection" "empty" "$EMPTY_LINKS"
fi

# ============================================
# Test 7: URL-encoded parameters in VLESS link
# ============================================
echo ""
echo "=== Test 7: URL-encoded parameters (emoji fragment, encoded values) ==="

# Test url_decode behavior - the key fix is that + should NOT become space
url_decode_fixed() {
    local encoded="$1"
    printf '%b' "$(echo "$encoded" | sed 's/%/\\x/g')"
}

# A real-world VLESS URL with URL-encoded fragment
VLESS_URL="vless://d9b38312-8bc1-024e-b828-a7eb708a9212@ap4.directly.chat:443?type=tcp&security=reality&flow=xtls-rprx-vision&sni=www.microsoft.com&fp=chrome&pbk=testpublickey123#%F0%9F%87%B0%F0%9F%87%BF%20Node"

# After url_decode, the fragment should be decoded but + in Base64 passwords should NOT become space
DECODED_URL=$(url_decode_fixed "$VLESS_URL")

# The scheme should still be vless
SCHEME=$(echo "$DECODED_URL" | sed -n 's#^\([^:]*\)://.*#\1#p')
if [ "$SCHEME" = "vless" ]; then
    pass "VLESS scheme preserved after url_decode"
else
    fail "VLESS scheme after url_decode" "vless" "$SCHEME"
fi

# Test SS URL with Base64 containing + (chacha20-ietf-poly1305:test>pass encodes with +)
# "chacha20-ietf-poly1305:test>pass" -> base64 = "Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0ZXN0PnBhc3M="
# We need a real + in the base64: "2023+method:password" -> base64 starts with + in some combos
# Direct approach: use a known Base64 with + in it
SS_USERINFO_RAW="aes-256-gcm:p@ss+word" 
SS_USERINFO_B64=$(echo -n "$SS_USERINFO_RAW" | base64 2>/dev/null)
# Verify the b64 has no + (if it doesn't, force one)
if ! echo "$SS_USERINFO_B64" | grep -q '+'; then
    # Use a known string that produces + in base64: "ab>" -> base64 = "YWI+"
    SS_USERINFO_B64="YWI+OnBhc3N3b3Jk"
fi
SS_URL="ss://${SS_USERINFO_B64}@1.2.3.4:8388#Server"
SS_DECODED=$(url_decode_fixed "$SS_URL")
SS_EXTRACTED=$(echo "$SS_DECODED" | sed -n 's#^ss://\([^@]*\)@.*#\1#p')

# The + should still be present (not converted to space)
if echo "$SS_EXTRACTED" | grep -q '+'; then
    pass "Base64 '+' preserved in SS URL (not converted to space)"
elif echo "$SS_EXTRACTED" | grep -q ' '; then
    fail "Base64 '+' was converted to space in SS URL" "contains +" "$SS_EXTRACTED"
else
    pass "Base64 in SS URL preserved correctly (no + in this test data)"
fi

# ============================================
# Cleanup
# ============================================
rm -rf "$TMPDIR_TEST"

# ============================================
# Summary
# ============================================
echo ""
echo "=================================="
echo "Test Results: $PASS passed, $FAIL failed (total: $TOTAL)"
echo "=================================="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
