#!/usr/bin/env bash
# Teacher tool: regenerate .github/tests.sha256 after changing official files.
# Run from the repository root: bash scripts/checksums.sh
set -euo pipefail
cd "$(dirname "$0")/.."
sha256sum internal/app/stage*_test.go internal/app/helpers_test.go internal/app/variants_test.go \
  scripts/score.py .github/workflows/ci.yml > .github/tests.sha256
echo "updated .github/tests.sha256"
