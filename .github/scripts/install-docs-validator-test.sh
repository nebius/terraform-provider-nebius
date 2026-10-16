#!/usr/bin/env bash
set -eu -o pipefail

installer="$(cd "$(dirname "$0")" && pwd)/install-docs-validator.sh"
directory=$(mktemp -d)
trap 'rm -rf "$directory"' EXIT
mkdir "$directory/bin"
cat > "$directory/bin/go" <<'GO'
#!/usr/bin/env bash
echo attempt >> "$ATTEMPTS"
count=$(wc -l < "$ATTEMPTS")
if [ "$count" -le "$FAILURES" ]; then
    echo "$FAILURE_MESSAGE"
    exit 1
fi
printf '#!/bin/sh\nexit 0\n' > "$GOBIN/tfplugindocs"
chmod +x "$GOBIN/tfplugindocs"
GO
printf '#!/bin/sh\nexit 0\n' > "$directory/bin/sleep"
chmod +x "$directory/bin/"*
export PATH="$directory/bin:$PATH"
export ATTEMPTS="$directory/attempts"

FAILURES=3 FAILURE_MESSAGE='stream error: stream ID 233; INTERNAL_ERROR; received from peer' \
    bash "$installer" "$directory/tools"
[ "$(wc -l < "$ATTEMPTS")" -eq 4 ]
[ -x "$directory/tools/tfplugindocs" ]

for message in 'compile error' 'certificate verify failed' 'TLS handshake timeout and checksum mismatch'; do
    rm "$ATTEMPTS"
    if FAILURES=4 FAILURE_MESSAGE="$message" bash "$installer" "$directory/tools"; then
        echo 'Installation must fail' >&2
        exit 1
    fi
    [ "$(wc -l < "$ATTEMPTS")" -eq 1 ]
done

rm "$ATTEMPTS"
if FAILURES=4 FAILURE_MESSAGE='TLS handshake timeout' bash "$installer" "$directory/tools"; then
    echo 'Installation must stop after four attempts' >&2
    exit 1
fi
[ "$(wc -l < "$ATTEMPTS")" -eq 4 ]
