#!/bin/bash
# Generate secure tokens for Reposilite authentication
# Creates admin, write, and read tokens

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}Reposilite Token Generation${NC}"
echo "============================"
echo ""

# Generate secure random tokens (64 hex characters = 32 bytes)
ADMIN_TOKEN=$(openssl rand -hex 32)
WRITE_TOKEN=$(openssl rand -hex 32)
READ_TOKEN=$(openssl rand -hex 32)

echo -e "${BLUE}Generated Reposilite Tokens:${NC}"
echo ""
echo -e "${YELLOW}Admin Token (full access):${NC}"
echo "  $ADMIN_TOKEN"
echo ""
echo -e "${YELLOW}Write Token (deploy artifacts):${NC}"
echo "  $WRITE_TOKEN"
echo ""
echo -e "${YELLOW}Read Token (pull dependencies):${NC}"
echo "  $READ_TOKEN"
echo ""

# Optionally write to .env file
echo -e "${BLUE}Update .env file?${NC}"
read -p "Append tokens to .env file? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    ENV_FILE="$(dirname "$0")/../.env"

    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${YELLOW}Creating new .env file...${NC}"
        cp "$(dirname "$0")/../.env.example" "$ENV_FILE"
    fi

    # Check if tokens already exist in .env
    if grep -q "REPOSILITE_ADMIN_TOKEN=" "$ENV_FILE"; then
        echo -e "${YELLOW}Warning: Tokens already exist in .env${NC}"
        read -p "Overwrite existing tokens? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted."
            exit 0
        fi

        # Update existing tokens
        sed -i.bak "s/^REPOSILITE_ADMIN_TOKEN=.*/REPOSILITE_ADMIN_TOKEN=$ADMIN_TOKEN/" "$ENV_FILE"
        sed -i.bak "s/^REPOSILITE_WRITE_TOKEN=.*/REPOSILITE_WRITE_TOKEN=$WRITE_TOKEN/" "$ENV_FILE"
        sed -i.bak "s/^REPOSILITE_READ_TOKEN=.*/REPOSILITE_READ_TOKEN=$READ_TOKEN/" "$ENV_FILE"
        rm "${ENV_FILE}.bak"
    else
        # Append tokens
        echo "" >> "$ENV_FILE"
        echo "# Reposilite Tokens (generated $(date))" >> "$ENV_FILE"
        echo "REPOSILITE_ADMIN_TOKEN=$ADMIN_TOKEN" >> "$ENV_FILE"
        echo "REPOSILITE_WRITE_TOKEN=$WRITE_TOKEN" >> "$ENV_FILE"
        echo "REPOSILITE_READ_TOKEN=$READ_TOKEN" >> "$ENV_FILE"
    fi

    echo -e "${GREEN}✅ Tokens written to $ENV_FILE${NC}"
fi

echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Update .env file with tokens above"
echo "2. Restart Reposilite:"
echo "   docker-compose restart reposilite"
echo "3. Tokens are automatically configured in Reposilite via environment variables"
echo ""
echo -e "${YELLOW}IMPORTANT: Save these tokens securely!${NC}"
echo "They will be used by Artifusion to authenticate to Reposilite."
