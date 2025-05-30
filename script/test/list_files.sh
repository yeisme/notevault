#!/bin/bash

echo "=== File List API Tests ==="

echo -e "\n1. List all files:"
curl -s "http://localhost:8888/api/v1/files/" | jq '.'

echo -e "\n2. List files with fileName=test2.md:"
curl -s "http://localhost:8888/api/v1/files/?fileName=test2.md" | jq '.'

echo -e "\n3. List files with fileName=test2.md and fileType=text/markdown:"
curl -s "http://localhost:8888/api/v1/files/?fileName=test2.md&fileType=text/markdown" | jq '.'

echo -e "\n4. List files with tags=test:"
curl -s "http://localhost:8888/api/v1/files/?tags=test&sortBy=name&order=asc" | jq '.'

echo -e "\n5. List files with pagination (page=1, pageSize=5):"
curl -s "http://localhost:8888/api/v1/files/?page=1&pageSize=5" | jq '.'

echo -e "\n6. File count summary:"
echo "Total files: $(curl -s "http://localhost:8888/api/v1/files/" | jq '.totalCount // 0')"
echo "Files with 'test' tag: $(curl -s "http://localhost:8888/api/v1/files/?tags=test" | jq '.totalCount // 0')"

echo -e "\nFile list tests completed!"
