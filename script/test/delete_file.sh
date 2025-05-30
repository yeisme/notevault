#!/bin/bash

TEST_FILE_1=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r .files[0].fileId)

TEST_FILE_2=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r .files[1].fileId)

echo "Testing with file ID: ${TEST_FILE_1} and ${TEST_FILE_2}"

curl -X DELETE "http://localhost:8888/api/v1/files/${TEST_FILE_2}" \
    -H 'Accept: application/json'
curl -X DELETE "http://localhost:8888/api/v1/files/${TEST_FILE_1}" \
    -H 'Accept: application/json'
