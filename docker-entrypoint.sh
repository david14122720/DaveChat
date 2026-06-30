#!/bin/sh
set -e

MARIADB_USER="${MARIADB_USER:-mysql}"
MARIADB_DATADIR="${MARIADB_DATADIR:-/var/lib/mysql}"
DB_NAME="${DB_NAME:-davechat}"
DB_USER="${DB_USER:-davechat}"
DB_PASS="${DB_PASS:-davechat_pass}"

# Initialize MariaDB data directory if first run
if [ ! -d "$MARIADB_DATADIR/mysql" ]; then
    echo "→ Initializing MariaDB data directory..."
    mariadb-install-db --user="$MARIADB_USER" --datadir="$MARIADB_DATADIR"
fi

# Ensure socket directory exists
mkdir -p /run/mysqld
chown "$MARIADB_USER" /run/mysqld

# Start MariaDB in background (explicit TCP port — Alpine defaults to port 0)
echo "→ Starting MariaDB..."
mariadbd --user="$MARIADB_USER" --datadir="$MARIADB_DATADIR" --port=3306 --bind-address=127.0.0.1 &
MARIADB_PID=$!

# Wait for MariaDB to be ready (busybox-compatible loop)
echo "→ Waiting for MariaDB..."
i=0
while [ $i -lt 30 ]; do
    if mariadb-admin ping --silent 2>/dev/null; then
        break
    fi
    i=$((i + 1))
    sleep 1
done

if ! mariadb-admin ping --silent 2>/dev/null; then
    echo "✗ MariaDB failed to start after 30s"
    exit 1
fi

echo "→ MariaDB ready, initializing database..."

mariadb -u root -e "
    CREATE DATABASE IF NOT EXISTS \`$DB_NAME\`
        CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    CREATE USER IF NOT EXISTS '$DB_USER'@'localhost'
        IDENTIFIED BY '$DB_PASS';
    GRANT ALL PRIVILEGES ON \`$DB_NAME\`.* TO '$DB_USER'@'localhost';
    FLUSH PRIVILEGES;
"

# Run schema
mariadb -u root "$DB_NAME" < /app/infra/mariadb/init.sql

echo "→ Database ready, starting DaveChat..."
exec "$@"
