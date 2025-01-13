#!/bin/sh
# wait-for-postgres.sh

set -e

until pg_isready -h localhost -p 5432 -U "$POSTGRES_USER"; do
  >&2 echo "PostgreSQL is unavailable - sleeping"
  sleep 1
done

>&2 echo "PostgreSQL is up - executing command" 