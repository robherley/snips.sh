#!/usr/bin/env bash
# Records the README tui demo against a throwaway local server that
# impersonates snips.sh (external addresses + an ssh host alias), seeded with
# demo files so the recording is reproducible.
#
# Usage: docs/tapes/record-tui.sh [output.gif]
set -euo pipefail

cd "$(dirname "$0")/../.."

if [[ ! -x bin/snips.sh ]]; then
  echo "bin/snips.sh not found; run 'just build' first" >&2
  exit 1
fi

OUT="${1:-tui.gif}"
DEMO_DIR="$(mktemp -d)"
DB="$DEMO_DIR/snips.db"
SERVER_PID=""
trap '[[ -n "$SERVER_PID" ]] && kill "$SERVER_PID" 2>/dev/null; rm -rf "$DEMO_DIR"' EXIT

SNIPS_DB_FILEPATH="$DB" \
SNIPS_SSH_INTERNAL="ssh://127.0.0.1:2224" \
SNIPS_HTTP_INTERNAL="http://127.0.0.1:8084" \
SNIPS_SSH_EXTERNAL="ssh://snips.sh:22" \
SNIPS_HTTP_EXTERNAL="https://snips.sh" \
  ./bin/snips.sh >"$DEMO_DIR/server.log" 2>&1 &
SERVER_PID=$!
sleep 1.5

# makes `ssh snips.sh` hit the demo server; the tape's hidden alias picks the
# path up from $SNIPS_DEMO_SSH_CONFIG
export SNIPS_DEMO_SSH_CONFIG="$DEMO_DIR/ssh_config"
cat >"$SNIPS_DEMO_SSH_CONFIG" <<'EOF'
Host snips.sh
  HostName 127.0.0.1
  Port 2224
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
  LogLevel ERROR
EOF

# upload stdin (extra args pass through to snips), echo the new file id
upload() {
  ssh -F "$SNIPS_DEMO_SSH_CONFIG" snips.sh "$@" 2>/dev/null |
    sed $'s/\x1b\\[[0-9;]*m//g' | grep -oE 'id: [A-Za-z0-9_-]+' | cut -d' ' -f2
}

# backdate <id> <sqlite time modifier, e.g. -3 days>
backdate() {
  sqlite3 "$DB" "UPDATE files
    SET created_at = strftime('%Y-%m-%d %H:%M:%f', 'now', '$2') || '+00:00',
        updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now', '$2') || '+00:00'
    WHERE id = '$1';"
}

# seed <age> [upload flags...] with the file content on stdin
seed() {
  local age="$1"
  shift
  backdate "$(upload "$@")" "$age"
}

seed '-6 days' <<'EOF'
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  created_at DATETIME NOT NULL,
  theme_color TEXT
);

CREATE TABLE files (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users (id),
  size INTEGER NOT NULL,
  content BLOB,
  private INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_files_user_id ON files (user_id);
EOF

seed '-3 days' <<'EOF'
services:
  snips:
    image: ghcr.io/robherley/snips.sh:latest
    restart: unless-stopped
    ports:
      - "2222:2222"
      - "8080:8080"
    volumes:
      - data:/data

volumes:
  data:
EOF

seed '-1 days' -- -private <<'EOF'
# release notes (draft)

## v0.4.0

- new file browser with fuzzy filtering
- signed urls for private files
- theme colors

## todo

- [ ] update screenshots
- [ ] changelog entry
- [ ] tag the release
EOF

seed '-2 hours' <<'EOF'
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from %s\n", r.Host)
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
EOF

vhs docs/tapes/tui.tape -o "$OUT"
echo "wrote $OUT"
