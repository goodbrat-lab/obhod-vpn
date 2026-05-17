#!/bin/bash
# Auto-publish script for Obhod to GitHub
# This script commits changes and publishes to GitHub

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}   Obhod Auto-Publish Script v1.1.1${NC}"
echo -e "${GREEN}==================================================${NC}"

# Check if we're in git repo
if [ ! -d .git ]; then
    echo -e "${RED}ERROR: Not a git repository${NC}"
    echo "Please run this script from the Obhod project root"
    exit 1
fi

# Check git status
if git status --porcelain | grep -q .; then
    echo -e "${YELLOW}Changes detected in repository${NC}"
    
    echo "Adding all changes..."
    git add .
    
    echo "Creating commit with security fixes..."
    git commit -m "🔒 Security fixes and improvements v1.1.1

🚨 Critical Security Fixes:
- Fix temporary file vulnerability symlink attack risk
- Fix proxy variable mismatch causing connection failures
- Add input validation to prevent UCI command injection

🛡️ Security Enhancements:
- Implement secure temp directory creation with trap cleanup
- Add cryptographically secure random ID generation
- Improve domain validation for RFC compliance
- Enhance error handling for network operations

🔧 Stability Improvements:
- Fix cron job creation/removal with error checking
- Add command availability checks before execution
- Improve log reading with timeout protection
- Better PID file handling with error checking

📦 Packaging:
- Sync version numbers across all files (v1.1.1)
- Add secure mktemp helper script
- Update configuration templates with new options

🧪 Testing:
- Improve test suite with secure temp handling
- Add validation for critical functions

This release addresses multiple security and stability issues
discovered during code review. All users should upgrade urgently."

else
    echo -e "${YELLOW}No changes to commit${NC}"
fi

# Get current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo -e "${GREEN}Current branch: $BRANCH${NC}"

# Check if remote exists
if git remote get-url origin >/dev/null 2>&1; then
    echo -e "${GREEN}Pushing to GitHub...${NC}"
    git push origin "$BRANCH" --tags
    echo -e "${GREEN}✅ Successfully pushed to GitHub${NC}"
else
    echo -e "${RED}No remote 'origin' found${NC}"
    echo "Please add your GitHub repository as remote:"
    echo "git remote add origin https://github.com/YOUR_USERNAME/obhod.git"
    exit 1
fi

# Show commit info
echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}Latest commit:${NC}"
git log -1 --oneline

echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}Repository published successfully!${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Visit your GitHub repository"
echo "2. Create a new release from master branch"
echo "3. Tag version v1.1.1"
echo "4. Upload built packages from dist/packages/"
echo -e "${GREEN}==================================================${NC}"