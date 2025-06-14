#!/usr/bin/env bash
# Update file metadata test script

# Set to display error messages
set -e

echo "=== Testing Update File Metadata API ==="

# Get a test file ID from existing files
TEST_FILE_ID=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r '.files[0].fileId // empty')

if [ -z "$TEST_FILE_ID" ] || [ "$TEST_FILE_ID" = "null" ]; then
    echo "No test files found. Please upload some test files first using file_upload.sh"
    exit 1
fi

echo "Testing with file ID: ${TEST_FILE_ID}"

# Get current file metadata to compare
echo -e "\n--- Getting current file metadata ---"
CURRENT_METADATA=$(curl -s "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}")
echo "Current metadata:"
echo "$CURRENT_METADATA" | jq .

CURRENT_VERSION=$(echo "$CURRENT_METADATA" | jq -r '.metadata.version')
echo "Current version: $CURRENT_VERSION"

# Test 1: Update file name only
echo -e "\n--- Test 1: Update file name only ---"
NEW_FILE_NAME="updated_test_$(date +%s).md"
curl -X PUT "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}" \
    -H "Authorization: Bearer your_token_here" \
    -H "Content-Type: application/json" \
    -d "{
        \"fileName\": \"${NEW_FILE_NAME}\",
        \"commitMessage\": \"Updated file name for testing\"
    }" | jq .

# Test 2: Update description only
echo -e "\n--- Test 2: Update description only ---"
NEW_DESCRIPTION="Updated description at $(date)"
curl -X PUT "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}" \
    -H "Authorization: Bearer your_token_here" \
    -H "Content-Type: application/json" \
    -d "{
        \"description\": \"${NEW_DESCRIPTION}\",
        \"commitMessage\": \"Updated description for testing\"
    }" | jq .

# Test 3: Update tags only
echo -e "\n--- Test 3: Update tags only ---"
curl -X PUT "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}" \
    -H "Authorization: Bearer your_token_here" \
    -H "Content-Type: application/json" \
    -d '{
        "tags": ["test", "updated", "metadata"],
        "commitMessage": "Updated tags for testing"
    }' | jq .

# Test 4: Update multiple fields at once
echo -e "\n--- Test 4: Update multiple fields at once ---"
MULTI_UPDATE_NAME="multi_update_test_$(date +%s).md"
MULTI_UPDATE_DESC="Multi-field update test at $(date)"
curl -X PUT "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}" \
    -H "Authorization: Bearer your_token_here" \
    -H "Content-Type: application/json" \
    -d "{
        \"fileName\": \"${MULTI_UPDATE_NAME}\",
        \"description\": \"${MULTI_UPDATE_DESC}\",
        \"tags\": [\"test\", \"multi-update\", \"final\"],
        \"commitMessage\": \"Multi-field update test\"
    }" | jq .

# Test 5: Get updated metadata to verify changes
echo -e "\n--- Test 5: Verify final metadata ---"
FINAL_METADATA=$(curl -s "http://localhost:8888/api/v1/files/metadata/${TEST_FILE_ID}")
echo "Final metadata:"
echo "$FINAL_METADATA" | jq .

FINAL_VERSION=$(echo "$FINAL_METADATA" | jq -r '.metadata.version')
echo "Final version: $FINAL_VERSION"
echo "Version increments: $((FINAL_VERSION - CURRENT_VERSION))"
