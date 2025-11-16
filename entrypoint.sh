#!/bin/sh
set -e

echo "Waiting for postgres..."
until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do
  sleep 1
done

echo "Running migrations..."
migrate -path ./migrations -database "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE" up

echo "Starting server..."
exec ./server
