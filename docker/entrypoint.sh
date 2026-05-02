#!/bin/sh
# Moon Panel container entrypoint.
#
# Maps the in-container "moon" user to the host UID/GID via PUID/PGID env vars,
# chowns the data volume, then drops privileges and execs the server.
# Idempotent across restarts: if PUID/PGID match an existing system entry
# (e.g. PGID=100 = "users" on alpine), we reuse that name instead of clashing.
#
# Defaults: PUID=1000, PGID=1000 (matches typical Linux desktop / cloud VMs).
# Synology: set PUID=1026 PGID=100. Find your values with `id` on the host.

set -eu

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"
DATA_DIR="${MOON_DATA_DIR:-/data}"

# Find an existing group with that GID, otherwise create "moon".
GROUP_NAME=$(awk -F: -v gid="$PGID" '$3 == gid { print $1; exit }' /etc/group)
if [ -z "$GROUP_NAME" ]; then
    addgroup -g "$PGID" moon
    GROUP_NAME=moon
fi

# Find an existing user with that UID, otherwise create "moon".
USER_NAME=$(awk -F: -v uid="$PUID" '$3 == uid { print $1; exit }' /etc/passwd)
if [ -z "$USER_NAME" ]; then
    adduser -u "$PUID" -G "$GROUP_NAME" -D -H -s /sbin/nologin moon
    USER_NAME=moon
fi

mkdir -p "$DATA_DIR"

# Chown by NUMERIC PUID:PGID rather than resolved USER_NAME:GROUP_NAME, so a
# corrupted /etc/passwd or a silent adduser/addgroup failure above still
# results in correctly-owned files (kernel only cares about numeric IDs;
# passwd/group are name lookup tables). Fail-fast on chown error: the
# server would otherwise crash later opening /data/jwt.key with an opaque
# permission-denied, and the container would restart-loop without an
# obvious cause. Better to bail here with a clear message.
echo "[entrypoint] PUID=$PUID PGID=$PGID DATA_DIR=$DATA_DIR (user=$USER_NAME group=$GROUP_NAME)"
if ! chown -R "$PUID:$PGID" "$DATA_DIR"; then
    echo "[entrypoint] FATAL: chown -R $PUID:$PGID $DATA_DIR failed" >&2
    echo "[entrypoint]   check the volume is writable and not mounted read-only" >&2
    exit 1
fi
echo "[entrypoint] chown'd $DATA_DIR to $PUID:$PGID; starting moon-panel"

exec su-exec "$USER_NAME:$GROUP_NAME" "$@"
