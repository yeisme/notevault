#!/bin/bash

FILE_1=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r .files[0].fileId)
FILE_NUMBER=$(curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq -r .files[0].version)
echo "Getting file metadata for file ID: ${FILE_1}"

curl --location "http://localhost:8888/api/v1/files/metadata/${FILE_1}" \
    --header 'Accept: application/json'

echo "Getting file metadata for file ID: ${FILE_1} with version_number: ${FILE_NUMBER}"

curl --location "http://localhost:8888/api/v1/files/metadata/${FILE_1}?version_number=${FILE_NUMBER}" \
    --header 'Accept: application/json'
