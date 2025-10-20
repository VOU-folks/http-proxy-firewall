#!/bin/env zsh

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if .upload file exists
if [ ! -f ".upload" ]; then
  echo -e "${RED}Error: .upload file not found${NC}"
  echo -e "${YELLOW}Please create .upload file from .upload.example${NC}"
  exit 1
fi

# Read configuration from .upload file
echo -e "${YELLOW}Reading configuration from .upload file...${NC}"

while IFS='=' read -r key value; do
  # Skip comments and empty lines
  [[ "$key" =~ ^#.*$ ]] && continue
  [[ -z "$key" ]] && continue

  # Trim whitespace
  key=$(echo "$key" | xargs)
  value=$(echo "$value" | xargs)

  case "$key" in
  server)
    REMOTE_HOST="$value"
    ;;
  user)
    REMOTE_USER="$value"
    ;;
  ssh-key)
    SSH_KEY="${value/#\~/$HOME}" # Expand ~ to $HOME
    ;;
  folder)
    REMOTE_PATH="$value"
    ;;
  esac
done <.upload

LOCAL_PATH="."

# Validate required configuration
if [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_USER" ] || [ -z "$SSH_KEY" ] || [ -z "$REMOTE_PATH" ]; then
  echo -e "${RED}Error: Missing required configuration in .upload file${NC}"
  echo -e "${YELLOW}Required fields: server, user, ssh-key, folder${NC}"
  exit 1
fi

echo -e "${YELLOW}Starting upload to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}${NC}"

# Check if SSH key exists
if [ ! -f "$SSH_KEY" ]; then
  echo -e "${RED}Error: SSH key not found at ${SSH_KEY}${NC}"
  exit 1
fi

# Ensure remote directory exists
echo -e "${YELLOW}Ensuring remote directory exists...${NC}"
ssh -i "$SSH_KEY" "${REMOTE_USER}@${REMOTE_HOST}" "mkdir -p ${REMOTE_PATH}" 2>/dev/null

if [ $? -ne 0 ]; then
  echo -e "${RED}Error: Failed to connect to remote server or create directory${NC}"
  exit 1
fi

# Sync files using rsync
# --exclude preserves .env and proxy-firewall.conf at destination
# -avz: archive mode, verbose, compress
# --delete: remove files at destination that don't exist in source (except excluded)
# -e: specify ssh with key
echo -e "${YELLOW}Syncing files...${NC}"

rsync -avz --delete \
  --exclude='.env' \
  --exclude='proxy-firewall.conf' \
  --exclude='.git/' \
  --exclude='.cache/' \
  --exclude='files/' \
  --exclude='log/' \
  -e "ssh -i ${SSH_KEY}" \
  "${LOCAL_PATH}/" \
  "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/"

if [ $? -eq 0 ]; then
  echo -e "${GREEN}Upload completed successfully!${NC}"
  echo -e "${YELLOW}Note: .env and proxy-firewall.conf at destination were preserved${NC}"
else
  echo -e "${RED}Error: Upload failed${NC}"
  exit 1
fi
