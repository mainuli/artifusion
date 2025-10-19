#!/bin/bash
# Generate htpasswd file for Docker registry:2 authentication
# Requires: apache2-utils (htpasswd command)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AUTH_DIR="$SCRIPT_DIR/../auth"
HTPASSWD_FILE="$AUTH_DIR/registry.htpasswd"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Docker Registry Authentication Setup${NC}"
echo "========================================"
echo ""

# Check if htpasswd is installed
if ! command -v htpasswd &> /dev/null; then
    echo -e "${RED}Error: htpasswd command not found${NC}"
    echo ""
    echo "Install apache2-utils:"
    echo "  Ubuntu/Debian: sudo apt-get install apache2-utils"
    echo "  macOS:         brew install httpd"
    echo "  Alpine:        apk add apache2-utils"
    exit 1
fi

# Create auth directory if it doesn't exist
mkdir -p "$AUTH_DIR"

# Default username
USERNAME="${1:-artifusion}"

# Check if htpasswd file exists
if [ -f "$HTPASSWD_FILE" ]; then
    echo -e "${YELLOW}Warning: htpasswd file already exists at:${NC}"
    echo "  $HTPASSWD_FILE"
    echo ""
    read -p "Overwrite existing file? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
    rm "$HTPASSWD_FILE"
fi

# Prompt for password or generate one
echo ""
read -p "Enter password for user '$USERNAME' (leave empty to generate): " PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    # Generate secure random password
    PASSWORD=$(openssl rand -base64 24)
    echo -e "${GREEN}Generated password:${NC} $PASSWORD"
    echo -e "${YELLOW}IMPORTANT: Save this password securely!${NC}"
    echo ""
fi

# Create htpasswd file with bcrypt encryption
htpasswd -Bbn "$USERNAME" "$PASSWORD" > "$HTPASSWD_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Success!${NC}"
    echo ""
    echo "Created htpasswd file at:"
    echo "  $HTPASSWD_FILE"
    echo ""
    echo "Credentials:"
    echo "  Username: $USERNAME"
    echo "  Password: $PASSWORD"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Add to .env file:"
    echo "   REGISTRY_USERNAME=$USERNAME"
    echo "   REGISTRY_PASSWORD=$PASSWORD"
    echo ""
    echo "2. Restart Docker registry:"
    echo "   docker-compose restart registry"
    echo ""
    echo "3. Test authentication:"
    echo "   docker login localhost:5000 -u $USERNAME -p $PASSWORD"
else
    echo -e "${RED}Failed to create htpasswd file${NC}"
    exit 1
fi
