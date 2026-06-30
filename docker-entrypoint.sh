#!/bin/sh
set -e

# Initialize MariaDB data directory if first run
if [ ! -d "/var/lib/mysql/mysql" ]; then
    echo "→ Initializing MariaDB data directory..."
    mariadb-install-db --user=mysql --datadir=/var/lib/mysql
fi

# Start MariaDB in background
echo "→ Starting MariaDB..."
mariadbd --user=mysql --datadir=/var/lib/mysql &
MARIADB_PID=$!

# Wait for MariaDB to be ready
for i in $(seq 30); do
    if mariadb-admin ping --silent 2>/dev/null; then
        break
    fi
    if [ "$i" = "30" ]; then
        echo "✗ MariaDB failed to start"
        exit 1
    fi
    sleep 1
done

echo "→ MariaDB ready, initializing database..."

# Create database and user
mariadb -u root -e "
    CREATE DATABASE IF NOT EXISTS davechat
        CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    CREATE USER IF NOT EXISTS 'davechat'@'localhost'
        IDENTIFIED BY 'davechat_pass';
    GRANT ALL PRIVILEGES ON davechat.* TO 'davechat'@'localhost';
    FLUSH PRIVILEGES;
"

# Run schema
mariadb -u root davechat < /app/infra/mariadb/init.sql

echo "→ Database ready, starting DaveChat..."

# Execute the main application
exec "$@"
