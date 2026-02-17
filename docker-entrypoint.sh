#!/bin/sh

set -e

echo "Starting deployment process..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "ERROR: DATABASE_URL is not set"
    exit 1
fi

# Run migrations
echo "Running database migrations..."
./migrator \
    -database-url="$DATABASE_URL" \
    -migrations-path=./migrations

if [ $? -eq 0 ]; then
    echo "Migrations completed successfully"
else
    echo "ERROR: Migrations failed"
    exit 1
fi

# Start the application
echo "Starting gRPC Auth application..."
exec ./grpc-auth --config=./config/prod.yaml
