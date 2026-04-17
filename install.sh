#!/usr/bin/env bash
set -euo pipefail

REPO="booscaaa/estrova"
BINARY="estrova"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get latest release tag
echo "Fetching latest release..."
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Could not determine latest release tag."
  exit 1
fi

echo "Installing ${BINARY} ${TAG} (${OS}/${ARCH})..."

FILENAME="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "${TMP}/${FILENAME}"
tar -xzf "${TMP}/${FILENAME}" -C "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "estrova installed to ${INSTALL_DIR}/${BINARY}"

# Ask for Strava credentials
echo ""
echo "To connect with Strava you need API credentials."
echo "Get them at: https://www.strava.com/settings/api"
echo "  - Application Name: anything (e.g. My Claude Coach)"
echo "  - Website: http://localhost"
echo "  - Authorization Callback Domain: localhost"
echo ""

read -rp "Enter your Strava Client ID: " CLIENT_ID </dev/tty
read -rsp "Enter your Strava Client Secret: " CLIENT_SECRET </dev/tty
echo ""

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
  echo ""
  echo "Credentials skipped. Run this manually when ready:"
  echo ""
  echo "  claude mcp add estrova ${INSTALL_DIR}/${BINARY} \\"
  echo "    -e STRAVA_CLIENT_ID=your_client_id \\"
  echo "    -e STRAVA_CLIENT_SECRET=your_client_secret"
  echo ""
  exit 0
fi

# Install estrova skills for Claude Code
SKILLS_DIR="$HOME/.claude/plugins/local/plugins/estrova/skills"
SKILLS_RAW="https://raw.githubusercontent.com/${REPO}/main/skills/estrova"

echo "Installing estrova skills..."
for skill in estrova-coach estrova-analysis estrova-plan estrova-resolve-conflicts; do
  mkdir -p "$SKILLS_DIR/$skill"
  curl -fsSL "${SKILLS_RAW}/${skill}/SKILL.md" -o "$SKILLS_DIR/$skill/SKILL.md"
done

SETTINGS="$HOME/.claude/settings.json"
  if [ -f "$SETTINGS" ]; then
    if ! grep -q '"estrova@local"' "$SETTINGS"; then
      python3 - "$SETTINGS" <<'PYEOF'
import json, sys
path = sys.argv[1]
with open(path) as f:
    s = json.load(f)
s.setdefault("enabledPlugins", {})["estrova@local"] = True
s.setdefault("extraKnownMarketplaces", {}).setdefault("local", {
    "source": {"source": "directory", "path": str(__import__('pathlib').Path.home() / ".claude/plugins/local")}
})
with open(path, "w") as f:
    json.dump(s, f, indent=4)
PYEOF
    fi
  else
    mkdir -p "$(dirname "$SETTINGS")"
    cat > "$SETTINGS" <<EOF
{
    "enabledPlugins": {
        "estrova@local": true
    },
    "extraKnownMarketplaces": {
        "local": {
            "source": {
                "source": "directory",
                "path": "${HOME}/.claude/plugins/local"
            }
        }
    }
}
EOF
  fi
echo "Skills installed!"

# Register MCP server in Claude Code
if command -v claude &>/dev/null; then
  echo "Registering estrova in Claude Code..."
  claude mcp add estrova "${INSTALL_DIR}/${BINARY}" \
    -e STRAVA_CLIENT_ID="$CLIENT_ID" \
    -e STRAVA_CLIENT_SECRET="$CLIENT_SECRET"
  echo ""
  echo "All done! Open Claude Code and run:"
  echo ""
  echo "  authenticate with strava"
  echo ""
else
  echo "Claude Code CLI not found. Add the server manually to ~/.claude/settings.json:"
  echo ""
  cat <<EOF
{
  "mcpServers": {
    "estrova": {
      "command": "${INSTALL_DIR}/${BINARY}",
      "env": {
        "STRAVA_CLIENT_ID": "${CLIENT_ID}",
        "STRAVA_CLIENT_SECRET": "${CLIENT_SECRET}"
      }
    }
  }
}
EOF
  echo ""
fi
