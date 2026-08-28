#!/bin/sh
# theboringoffice — TheBoringMajdoor attribution hook (git commit-msg).
#
# Auto-installed by the office into the repo it boots in when attribution is
# on (the default); scripts/install-majdoor-hook.sh covers repos the office
# never boots in. Either way git invokes it as:
#   .git/hooks/commit-msg <message-file>
#
# Every commit gets exactly one
#   Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
# trailer: skipped when a trailer carrying that email is already present
# (matched case-insensitively), joined onto an existing trailer block with no
# blank line, otherwise paragraph-broken with one. Running it twice changes
# nothing. POSIX sh; grep/sed are the only text tools.
set -eu

MSG_FILE="${1:?usage: commit-msg <message-file>}"

TRAILER="Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>"

# Already stamped? ANY trailer line (Token: value) carrying our email, in any
# case, means this message is done — never stamp twice.
if grep -qiE '^[A-Za-z][A-Za-z0-9-]*[[:space:]]*:.*themajdoor@theboring\.name' "$MSG_FILE"; then
    exit 0
fi

TMP_MSG="${MSG_FILE}.majdoor.$$"
trap 'rm -f "$TMP_MSG"' EXIT

# Strip trailing blank lines first, so the trailer hugs the end of the
# message instead of drifting off it. (Classic portable sed: trailing blanks
# accumulate in the pattern space and are dropped only at EOF; interior
# blank lines survive untouched.)
sed -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$MSG_FILE" > "$TMP_MSG"

# Trailer-block etiquette: when the message already ENDS in a trailer block
# (its last line is "Token: value") ours joins that block directly; anything
# else gets one blank line first, opening a fresh trailer paragraph.
if [ -s "$TMP_MSG" ] \
    && sed -n '$p' "$TMP_MSG" | grep -qE '^[A-Za-z][A-Za-z0-9-]*[[:space:]]*:'; then
    printf '%s\n' "$TRAILER" >> "$TMP_MSG"
else
    printf '\n%s\n' "$TRAILER" >> "$TMP_MSG"
fi

mv -f "$TMP_MSG" "$MSG_FILE"
