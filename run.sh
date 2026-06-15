#!/usr/bin/env bash
# Run the SMF Go port: write a config + admin account on first run, then serve.
# Re-running is safe — setup steps are skipped if already done.
#
#   ./run.sh                        # defaults: admin / admin / admin@example.com
#   ADMIN_USER=alex ADMIN_PASS=hunter2 ADMIN_EMAIL=me@x.com ./run.sh
set -euo pipefail

cd "$(dirname "$0")"

CONFIG="${CONFIG:-smf.conf}"
DB="${DB:-./smf.sqlite}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-admin}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"

# 1. Config (init refuses to overwrite, so only run it when missing).
if [ ! -f "$CONFIG" ]; then
	go run ./cmd/smf init -config "$CONFIG"
fi

# 2. Admin account + DB schema (only on first run, before the DB exists).
if [ ! -f "$DB" ]; then
	go run ./cmd/smf admin-create -config "$CONFIG" \
		-user "$ADMIN_USER" -password "$ADMIN_PASS" -email "$ADMIN_EMAIL"
	echo "admin login: $ADMIN_USER / $ADMIN_PASS"
fi

# 3. Serve.
exec go run ./cmd/smf -config "$CONFIG"
