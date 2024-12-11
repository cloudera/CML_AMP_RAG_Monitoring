#!/usr/bin/env bash
set -eo pipefail

if [ -n "$LOCAL" ]; then
  echo "LOCAL is set, not refreshing project"
  exit 0
fi

# Fetch the latest changes from origin without merging
git fetch origin

# Get the current branch name
current_branch=$(git rev-parse --abbrev-ref HEAD)

# Check if there are any new commits on the remote branch
if git log HEAD..origin/"$current_branch" --oneline | grep .; then
    echo "There are updates on origin/$current_branch"
    exit 1
else
    echo "No changes on origin/$current_branch"
    exit 0
fi
