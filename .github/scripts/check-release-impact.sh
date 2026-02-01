#!/bin/bash

# Check release impact based on PR title
# Usage: ./check-release-impact.sh "PR_TITLE" "PR_NUMBER"

set -e

PR_TITLE="$1"
PR_NUMBER="$2"

if [ -z "$PR_TITLE" ] || [ -z "$PR_NUMBER" ]; then
    echo "Usage: $0 'PR_TITLE' 'PR_NUMBER'"
    exit 1
fi

# Determine release type
RELEASE_TYPE="none"
EMOJI="📝"

if [[ "$PR_TITLE" =~ ^feat!: ]]; then
    RELEASE_TYPE="major"
    EMOJI="💥"
elif [[ "$PR_TITLE" =~ ^fix!: ]]; then
    RELEASE_TYPE="major"
    EMOJI="💥"
elif [[ "$PR_TITLE" =~ ^feat: ]]; then
    RELEASE_TYPE="minor"
    EMOJI="✨"
elif [[ "$PR_TITLE" =~ ^fix: ]]; then
    RELEASE_TYPE="patch"
    EMOJI="🐛"
fi

# Generate message
case $RELEASE_TYPE in
    "major")
        MESSAGE="$EMOJI **MAJOR RELEASE** - This PR will trigger a major version bump (breaking changes)"
        ;;
    "minor")
        MESSAGE="$EMOJI **MINOR RELEASE** - This PR will trigger a minor version bump (new features)"
        ;;
    "patch")
        MESSAGE="$EMOJI **PATCH RELEASE** - This PR will trigger a patch version bump (bug fixes)"
        ;;
    *)
        MESSAGE="📝 **NO RELEASE** - This PR will not trigger a release"
        ;;
esac

# Create comment body
COMMENT_BODY="## Release Impact

$MESSAGE

**PR Title:** \`$PR_TITLE\`"

if [ "$RELEASE_TYPE" != "none" ]; then
    COMMENT_BODY="$COMMENT_BODY

🚀 A new release will be created when this PR is merged to main."
fi

# Output for GitHub Actions
echo "release_type=$RELEASE_TYPE" >> $GITHUB_OUTPUT
echo "comment_body<<EOF" >> $GITHUB_OUTPUT
echo "$COMMENT_BODY" >> $GITHUB_OUTPUT
echo "EOF" >> $GITHUB_OUTPUT
