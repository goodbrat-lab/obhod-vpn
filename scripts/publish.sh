#!/bin/bash
# Publish helper for Obhod
# This script validates the release repository and pushes only when explicitly invoked.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get version dynamically from constants.sh
SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPTS_DIR")"
VERSION=$(grep "OBHOD_VERSION=" "$BASE_DIR/obhod-core/files/usr/lib/constants.sh" | cut -d'\"' -f2)

echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}   Obhod Publish Helper v${VERSION}${NC}"
echo -e "${GREEN}==================================================${NC}"

# Check if we're in git repo
if [ ! -d .git ]; then
    echo -e "${RED}ERROR: Not a git repository${NC}"
    echo "Please run this script from the Obhod project root"
    exit 1
fi

echo "Checking release repository contents..."
./scripts/prepare_release_repo.sh

if [ -d dist/packages/usr ]; then
    echo -e "${RED}ERROR: dist/packages/usr still exists after cleanup${NC}"
    exit 1
fi

if ! ls dist/packages/luci-app-obhod_${VERSION}-1_all.ipk >/dev/null 2>&1; then
    echo -e "${RED}ERROR: LuCI release package is missing (expected version ${VERSION}-1)${NC}"
    exit 1
fi

if ! ls dist/packages/obhod_${VERSION}-1_*.ipk >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Core release packages are missing (expected version ${VERSION}-1)${NC}"
    exit 1
fi

if git status --porcelain | grep -q .; then
    echo -e "${YELLOW}Repository has uncommitted changes.${NC}"
    echo "Review them before pushing: git status && git diff"
else
    echo -e "${GREEN}Working tree is clean.${NC}"
fi

# Get current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo -e "${GREEN}Current branch: $BRANCH${NC}"

# Check if remote exists
if git remote get-url origin >/dev/null 2>&1; then
    echo -e "${GREEN}Remote 'origin' is configured.${NC}"
    echo "Push manually when ready: git push origin $BRANCH --tags"
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
echo -e "${GREEN}Publish preparation completed.${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review git status and git diff"
echo "2. Commit release changes with your chosen message"
echo "3. Push branch and tags manually"
echo "4. Create GitHub release for v${VERSION}"
echo "5. Upload packages from dist/packages/"
echo -e "${GREEN}==================================================${NC}"
