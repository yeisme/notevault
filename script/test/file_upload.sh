#!/usr/bin/env bash
# Upload markdown files to the server
curl -X POST "http://localhost:8888/api/v1/files/upload" \
    -H "Authorization: Bearer your_token_here" \
    -F "file=@test_file/test.md" \
    -F "fileName=test.md" \
    -F "fileType=text/markdown" \
    -F "description=This is a test file" \
    -F "tags=test,document" \
    -F "commitMessage=Initial upload"

curl -X POST "http://localhost:8888/api/v1/files/upload" \
    -H "Authorization: Bearer your_token_here" \
    -F "file=@test_file/test2.md" \
    -F "fileName=test2.md" \
    -F "fileType=text/markdown" \
    -F "description=This is a test file" \
    -F "tags=test,document" \
    -F "commitMessage=Initial upload"
