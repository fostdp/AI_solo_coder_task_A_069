#!/bin/bash

set -e

MINIO_ENDPOINT="http://minio:9000"
MINIO_ALIAS="simminio"
MINIO_USER="simadmin"
MINIO_PASSWORD="sim123456"
BUCKET_NAME="sim-scenario"
MAX_RETRIES=30
RETRY_INTERVAL=2

echo "Waiting for MinIO to be ready..."

retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    if curl -sf "${MINIO_ENDPOINT}/minio/health/live" > /dev/null 2>&1; then
        echo "MinIO is ready."
        break
    fi
    retry_count=$((retry_count + 1))
    echo "Attempt ${retry_count}/${MAX_RETRIES} - MinIO not ready, retrying in ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retry_count -eq $MAX_RETRIES ]; then
    echo "Error: MinIO did not become ready within the timeout."
    exit 1
fi

mc alias set ${MINIO_ALIAS} ${MINIO_ENDPOINT} ${MINIO_USER} ${MINIO_PASSWORD}

echo "Creating bucket: ${BUCKET_NAME}"
if mc ls ${MINIO_ALIAS}/${BUCKET_NAME} > /dev/null 2>&1; then
    echo "Bucket ${BUCKET_NAME} already exists."
else
    mc mb ${MINIO_ALIAS}/${BUCKET_NAME}
    echo "Bucket ${BUCKET_NAME} created."
fi

echo "Setting bucket policy for public read access..."
mc anonymous set download ${MINIO_ALIAS}/${BUCKET_NAME}

echo "Uploading sample placeholder files..."

echo "placeholder" | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/videos/README.txt
echo "placeholder" | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/can_logs/README.txt
echo "placeholder" | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/frames/README.txt
echo "placeholder" | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/exports/README.txt
echo "placeholder" | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/thumbnails/README.txt

echo '{"bucket":"sim-scenario","description":"Autonomous Driving Simulation Test Scenario Library","version":"1.0"}' | mc pipe ${MINIO_ALIAS}/${BUCKET_NAME}/meta/bucket_info.json

echo "MinIO initialization complete."
