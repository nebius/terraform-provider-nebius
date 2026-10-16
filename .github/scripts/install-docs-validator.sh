#!/usr/bin/env bash
set -eu -o pipefail

export GOBIN="$1"
mkdir -p "$GOBIN"
log=$(mktemp)
trap 'rm -f "$log"' EXIT
deadline=$(($(date +%s) + 300))
attempt=0

while true; do
    remaining=$((deadline - $(date +%s)))
    if [ "$remaining" -le 0 ]; then
        exit 124
    fi
    if timeout --kill-after=5s "${remaining}s" go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest > "$log" 2>&1; then
        cat "$log"
        exit 0
    else
        status=$?
    fi
    cat "$log"
    if [ "$status" -ge 128 ] || [ "$status" -eq 124 ] || [ "$attempt" -ge 3 ] ||
        grep -Eqi 'certificate|publickey|checksum mismatch|401 Unauthorized|403 Forbidden' "$log" ||
        ! grep -Eq 'TLS handshake timeout|i/o timeout|connection reset by peer|Temporary failure in name resolution|http2: server sent GOAWAY|stream error: stream ID [0-9]+; (CANCEL|INTERNAL_ERROR); received from peer' "$log"; then
        exit "$status"
    fi
    case "$attempt" in
        0) delay=2 ;;
        1) delay=5 ;;
        2) delay=10 ;;
    esac
    attempt=$((attempt + 1))
    remaining=$((deadline - $(date +%s)))
    if [ "$remaining" -le 0 ]; then exit 124; fi
    if [ "$delay" -gt "$remaining" ]; then delay=$remaining; fi
    echo "Network failure during docs validator installation. Retry $attempt/3 in $delay seconds."
    sleep "$delay"
done
