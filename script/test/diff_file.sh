#!/bin/bash

# Simple file diff test script
# This script tests file version diff API with existing files

set -e

SERVER_URL="http://localhost:8888"
AUTH_TOKEN="your_token_here"

echo "=== Simple File Version Diff Test ==="

# Get a test file ID from existing files
echo "Getting existing test file..."
TEST_FILE_ID=$(curl -s "${SERVER_URL}/api/v1/files/?tags=test&sortBy=name&order=asc" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" | jq -r '.files[0].fileId // empty')

if [ -z "$TEST_FILE_ID" ] || [ "$TEST_FILE_ID" = "null" ]; then
    echo "❌ No test files found. Please upload some test files first using file_upload.sh"
    exit 1
fi

echo "✅ Using file ID: ${TEST_FILE_ID}"
echo ""

# Get version list
echo "Getting version list..."
VERSIONS=$(curl -s "${SERVER_URL}/api/v1/files/${TEST_FILE_ID}/versions" \
    -H "Authorization: Bearer ${AUTH_TOKEN}")

echo "Available versions:"
echo "$VERSIONS" | jq '.versions[] | {version: .version, commitMessage: .commitMessage, createdAt: .createdAt}'
echo ""

# Extract version numbers
VERSION_COUNT=$(echo "$VERSIONS" | jq '.versions | length')
if [ "$VERSION_COUNT" -lt 2 ]; then
    echo "❌ Need at least 2 versions for diff comparison. Found: $VERSION_COUNT"
    echo "Upload a new version of this file first."
    exit 1
fi

FIRST_VERSION=$(echo "$VERSIONS" | jq -r '.versions[0].version')
SECOND_VERSION=$(echo "$VERSIONS" | jq -r '.versions[1].version')

echo "Comparing version ${FIRST_VERSION} with version ${SECOND_VERSION}..."

# Call diff API
curl -s "${SERVER_URL}/api/v1/files/${TEST_FILE_ID}/versions/diff?baseVersion=${FIRST_VERSION}&targetVersion=${SECOND_VERSION}" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" | jq '.'

echo ""
echo "✅ File diff test completed!"
