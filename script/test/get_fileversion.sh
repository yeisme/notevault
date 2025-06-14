#!/bin/bash

TEST_FILE_1=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r .files[0].fileId)
echo "Testing with file ID: ${TEST_FILE_1}"

curl -X GET "http://localhost:8888/api/v1/files/${TEST_FILE_1}/versions" \
    -H "Authorization: Bearer your_token_here" | jq
