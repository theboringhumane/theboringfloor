#!/usr/bin/env bash
# Compatibility entry point for current documentation captures.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec node "$SCRIPT_DIR/product-shots.mjs" --docs
