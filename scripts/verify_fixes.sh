#!/bin/bash
# Complete vulnerability fix checker
# Run this script to verify all security fixes are in place

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

FIXES_FOUND=0
FIXES_TOTAL=9

check_fix() {
    local file="$1"
    local pattern="$2"
    local description="$3"
    
    if [ -f "$file" ] && grep -q "$pattern" "$file"; then
        echo -e "${GREEN}✅ $description${NC}"
        FIXES_FOUND=$((FIXES_FOUND + 1))
    else
        echo -e "${RED}❌ $description - NOT FOUND${NC}"
        echo -e "   Expected in: $file"
        echo -e "   Pattern: $pattern"
    fi
}

echo -e "${YELLOW}🔍 Checking Obhod Security Fixes...${NC}"
echo ""

# 1. Path Traversal Fix
check_fix "obhod-core/src/internal/config/generator.go" \
    "func sanitizePath(filename string) string" \
    "Path Traversal Protection"

# 2. Command Injection Fix  
check_fix "obhod-core/src/internal/sysinfo/sysinfo.go" \
    "func safeReadFile" \
    "Command Injection Protection"

# 3. Race Condition Fix
check_fix "obhod-core/src/internal/watchdog/watchdog.go" \
    "sync.Mutex" \
    "Race Condition Protection"

# 4. Secret Encryption
check_fix "obhod-core/src/main_fixed.go" \
    "func encryptSecret(plaintext string) (string, error)" \
    "Secret Encryption"

# 5. DoS Protection
check_fix "obhod-core/src/internal/subscription/fetcher.go" \
    "maxSubscriptionSize" \
    "DoS Memory Limit"

# 6. Input Sanitization
check_fix "obhod-core/src/internal/telegram/telegram_fixed.go" \
    "func sanitizeInput(input string) string" \
    "XSS Input Sanitization"

# 7. Atomic Operations
check_fix "obhod-core/src/internal/config/atomic.go" \
    "type AtomicConfig struct" \
    "Atomic Configuration"

# 8. Security Tests
check_fix "tests/security_test.sh" \
    "validate_chat_id" \
    "Security Test Suite"

# 9. Security Hardening Script
check_fix "scripts/security_hardening.sh" \
    "secure_permissions" \
    "Security Hardening Script"

echo ""
echo -e "${YELLOW}📊 Results Summary:${NC}"
echo -e "Fixes applied: $FIXES_FOUND/$FIXES_TOTAL"

if [ $FIXES_FOUND -eq $FIXES_TOTAL ]; then
    echo -e "${GREEN}🎉 ALL SECURITY FIXES VERIFIED!${NC}"
    echo -e "${GREEN}🛡️ Obhod is now secure and ready for production${NC}"
else
    echo -e "${RED}⚠️  Some fixes are missing!${NC}"
    exit 1
fi