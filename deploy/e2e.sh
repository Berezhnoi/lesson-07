#!/usr/bin/env bash
# End-to-end lab check: the seeded admin logs into the web UI on K3s, creates a
# user via POST /register, and the confirmation email must appear in Floci SES.
# Usage: ADMIN_PASSWORD=... ./e2e.sh http://localhost:30080 http://localhost:4566
set -euo pipefail

APP=${1:?app base URL}
FLOCI=${2:?floci base URL}
ADMIN_EMAIL=${ADMIN_EMAIL:-admin@movies.local}
: "${ADMIN_PASSWORD:?ADMIN_PASSWORD must be set}"
NEW_EMAIL="lab.user.$(date +%s)@example.com"
NEW_PASSWORD="$(openssl rand -hex 12)"

jar=$(mktemp)
trap 'rm -f "$jar"' EXIT
origin=(-H "Origin: $APP")

csrf() { curl -s -b "$jar" -c "$jar" "$APP$1" | sed -n 's/.*<meta name="csrf-token" content="\([^"]*\)".*/\1/p'; }

token=$(csrf /login)
status=$(curl -s -o /dev/null -w '%{http_code}' -b "$jar" -c "$jar" "${origin[@]}" \
	--data-urlencode "email=$ADMIN_EMAIL" --data-urlencode "password=$ADMIN_PASSWORD" \
	--data-urlencode "csrf_token=$token" "$APP/login")
echo "admin login            -> HTTP $status (want 303)"
[ "$status" = 303 ]

token=$(csrf /register)
result=$(curl -s -o /dev/null -w '%{http_code} %{redirect_url}' -b "$jar" -c "$jar" "${origin[@]}" \
	--data-urlencode "email=$NEW_EMAIL" --data-urlencode "name=Lab User" \
	--data-urlencode "password=$NEW_PASSWORD" --data-urlencode "csrf_token=$token" "$APP/register")
echo "admin creates user     -> HTTP $result"
[ "${result%% *}" = 303 ]

for _ in $(seq 1 45); do
	if curl -s "$FLOCI/_aws/ses" | grep -q "\"$NEW_EMAIL\""; then
		echo "SES confirmation email -> found for $NEW_EMAIL"
		curl -s "$FLOCI/_aws/ses" | python3 -c '
import json, sys
to = sys.argv[1]
for m in json.load(sys.stdin)["messages"]:
    if to in m["Destination"]["ToAddresses"]:
        print("  From:", m["Source"]); print("  To:  ", to); print("  Subj:", m["Subject"])
' "$NEW_EMAIL"
		exit 0
	fi
	sleep 2
done
echo "SES confirmation email -> NOT found for $NEW_EMAIL" >&2
exit 1
