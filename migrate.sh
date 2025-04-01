#!/bin/sh

for cmd in awk soda ; do
  if ! command -v "$cmd" > /dev/null 2>&1 ; then
    >&2 echo "command '$cmd' is required."
    exit 1
  fi
done

# make sure we are in the right folder
scriptdir="$( cd "$( dirname "$0" )" && pwd )"
cd "${scriptdir}"

set -euo pipefail
trap 'exit' SIGINT SIGQUIT SIGTSTP SIGTERM SIGHUP

migrate() {
	if [ -f config/database.yml ]; then
		soda -c config/database.yml create >/dev/null 2>&1 || true
		soda -c config/database.yml migrate up 2>&1
	else
		soda create >/dev/null 2>&1 || true
		soda migrate up 2>&1
	fi
}

: "${DATABASE_HOST:?variable not set or empty}"
DATABASE_PORT=${DATABASE_PORT:-5432}

echo "[INFO] Waiting for Postgres to start..."
until nc -w 1 -z ${DATABASE_HOST} ${DATABASE_PORT}; do sleep 1; done
echo "[INFO] Postgres is up"

# sleep one more second to prevent from "pq: the database system is starting up" issue
sleep 1

migrate | awk '{ print "[INFO]", $0 }'
