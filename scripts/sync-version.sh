#!/bin/bash

# Sync version between .versionrc and version.go
# Usage: ./scripts/sync-version.sh

set -e

if [ ! -f .versionrc ]; then
    echo "❌ .versionrc file not found"
    exit 1
fi

VERSION=$(cat .versionrc | tr -d '[:space:]')

if [ -z "$VERSION" ]; then
    echo "❌ Version is empty in .versionrc"
    exit 1
fi

# Validate semantic version format
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo "❌ Invalid version format: $VERSION"
    echo "Expected format: x.x.x or x.x.x-suffix"
    exit 1
fi

# Update version.go
cat > version.go << EOF
package ripple

// Version represents the current version of the Ripple Go SDK
const Version = "$VERSION"
EOF

echo "✅ Version synced: $VERSION"
echo "Updated files:"
echo "  - .versionrc: $VERSION"
echo "  - version.go: const Version = \"$VERSION\""
