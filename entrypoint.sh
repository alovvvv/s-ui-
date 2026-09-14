#!/bin/sh
set -e

DB_FOLDER="${SUI_DB_FOLDER:-/app/db}"
DB_PATH="${DB_FOLDER}/s-ui.db"

if [ -f "$DB_PATH" ]; then
	./sui migrate
fi

# Run unprivileged when the operator asks for it.
#
# The default stays root because a TUN inbound needs CAP_NET_ADMIN in the
# process's permitted set and a panel port below 1024 needs
# CAP_NET_BIND_SERVICE; neither survives dropping to an ordinary user, so
# making this the default would break those deployments with no warning.
#
# With SUI_UID set, the data directory is handed over first -- it is normally a
# volume owned by root, and a panel that cannot write its own database is worse
# than one running with more privilege than it needs.
if [ -n "$SUI_UID" ]; then
	SUI_GID="${SUI_GID:-$SUI_UID}"
	if [ "$(id -u)" != "0" ]; then
		echo "entrypoint: SUI_UID is set but this container is not running as root; ignoring" >&2
	else
		mkdir -p "$DB_FOLDER"
		chown -R "${SUI_UID}:${SUI_GID}" "$DB_FOLDER" /app/sui /app/libcronet.so 2>/dev/null || true
		echo "entrypoint: dropping to uid ${SUI_UID}:${SUI_GID}"
		exec su-exec "${SUI_UID}:${SUI_GID}" ./sui
	fi
fi

exec ./sui
