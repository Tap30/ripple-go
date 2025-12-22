#!/bin/bash

# Extract version from branch name for release automation
# Usage: ./scripts/extract-version.sh <branch_name>

set -e

BRANCH_NAME="$1"

if [ -z "$BRANCH_NAME" ]; then
    echo "Usage: $0 <branch_name>"
    exit 1
fi

echo "Branch Name: $BRANCH_NAME"

# Check if branch matches release/x.x.x or release/x.x.x-suffix pattern
if [[ $BRANCH_NAME =~ ^release/([0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?)$ ]]; then
    VERSION="${BASH_REMATCH[1]}"
    
    # Set GitHub Actions outputs if running in CI
    if [ -n "$GITHUB_OUTPUT" ]; then
        echo "version=v$VERSION" >> $GITHUB_OUTPUT
        echo "should_release=true" >> $GITHUB_OUTPUT
    fi
    
    echo "✅ Match found!"
    echo "Extracted version: v$VERSION"
    echo "Should release: true"
else
    # Set GitHub Actions outputs if running in CI
    if [ -n "$GITHUB_OUTPUT" ]; then
        echo "should_release=false" >> $GITHUB_OUTPUT
    fi
    
    echo "❌ No match"
    echo "Branch name does not match release pattern"
    echo "Expected format: release/x.x.x or release/x.x.x-suffix"
    echo "Should release: false"
fi
