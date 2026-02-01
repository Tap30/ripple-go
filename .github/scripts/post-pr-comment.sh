#!/bin/bash

# Post or update PR comment with release impact
# Usage: ./post-pr-comment.sh

set -e

COMMENT_BODY="$1"
PR_NUMBER="$2"

if [ -z "$COMMENT_BODY" ] || [ -z "$PR_NUMBER" ]; then
    echo "Usage: $0 'COMMENT_BODY' 'PR_NUMBER'"
    exit 1
fi

REPO_OWNER=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f1)
REPO_NAME=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f2)

# Check if comment already exists
EXISTING_COMMENT_ID=$(gh api "repos/$REPO_OWNER/$REPO_NAME/issues/$PR_NUMBER/comments" \
    --jq '.[] | select(.user.login == "github-actions[bot]" and (.body | contains("## Release Impact"))) | .id' \
    | head -n1)

if [ -n "$EXISTING_COMMENT_ID" ]; then
    # Update existing comment
    gh api "repos/$REPO_OWNER/$REPO_NAME/issues/comments/$EXISTING_COMMENT_ID" \
        -X PATCH \
        -f body="$COMMENT_BODY" > /dev/null
    echo "Updated existing PR comment"
else
    # Create new comment
    gh api "repos/$REPO_OWNER/$REPO_NAME/issues/$PR_NUMBER/comments" \
        -f body="$COMMENT_BODY" > /dev/null
    echo "Created new PR comment"
fi
