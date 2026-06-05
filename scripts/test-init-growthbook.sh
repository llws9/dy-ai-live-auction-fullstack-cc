#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/scripts/init-growthbook.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_file() {
  local file=$1
  [[ -f "$file" ]] || fail "$file must exist"
}

assert_contains() {
  local pattern=$1
  local message=$2

  if ! rg -q "$pattern" "$SCRIPT"; then
    fail "$message"
  fi
}

assert_file "$SCRIPT"
assert_contains 'set -euo pipefail' "init-growthbook.sh must fail closed"
assert_contains 'mongosh --quiet "\$GROWTHBOOK_DB" --eval "\$script"' "init-growthbook.sh must run mongosh non-interactively"
assert_contains 'GROWTHBOOK_CLIENT_KEY:=dev-client-key' "init-growthbook.sh must keep the repo default client key"
assert_contains 'FEATURE_KEY:=live-start-popup-visibility' "init-growthbook.sh must initialize the live start popup experiment"
assert_contains 'ensure_growthbook_first_user' "init-growthbook.sh must bootstrap the first GrowthBook admin when needed"
assert_contains 'updateOne' "init-growthbook.sh must be idempotent through Mongo upserts"
assert_contains 'upsert: true' "init-growthbook.sh must not duplicate GrowthBook records on repeated runs"
assert_contains 'control' "init-growthbook.sh must define the control variation"
assert_contains 'treatment' "init-growthbook.sh must define the treatment variation"
assert_contains 'weight: 0.5' "init-growthbook.sh must keep 50/50 traffic split"
assert_contains 'curl -fsS "\$GROWTHBOOK_API_URL/api/features/\$GROWTHBOOK_CLIENT_KEY"' "init-growthbook.sh must verify the public SDK payload"
assert_contains 'jq -e' "init-growthbook.sh must fail if the expected feature is not visible in the public payload"

echo "init GrowthBook script checks passed"
