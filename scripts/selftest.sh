#!/usr/bin/env bash
# Safe self-test: no apply against real home skill trees.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="$ROOT/agent-skills-manager"
echo "== build =="
go build -o "$BIN" ./cmd/agent-skills-manager
test -x "$BIN"
file "$BIN"

pass=0
fail=0
check() {
  local name="$1" expect="$2"
  shift 2
  set +e
  out="$("$@" 2>&1)"
  code=$?
  set -e
  if [[ "$code" -eq "$expect" ]]; then
    echo "PASS  $name (exit $code)"
    pass=$((pass+1))
  else
    echo "FAIL  $name (exit $code want $expect)"
    echo "$out" | tail -20
    fail=$((fail+1))
  fi
}

echo "== go test =="
go test ./...

echo "== CLI =="
check version 0 "$BIN" version
check help 0 "$BIN" help
check scan 0 "$BIN" scan
check plan 0 "$BIN" plan
check doctor 0 "$BIN" doctor
check outdated 0 "$BIN" outdated
check unknown 2 "$BIN" nope
# non-TTY apply without --yes must fail
check apply-refuse 1 bash -c "echo | '$BIN' apply --strategy skip >/dev/null"

# alias binary name build
ASM="$ROOT/bin/asm"
mkdir -p "$ROOT/bin"
go build -o "$ROOT/bin/agent-skills-manager" ./cmd/agent-skills-manager
ln -sfn "$ROOT/bin/agent-skills-manager" "$ASM"
check asm-version 0 "$ASM" version

echo "== plan json =="
"$BIN" plan --out /tmp/agent-skills-manager-selftest-plan.json >/dev/null
python3 - <<'PY'
import json,sys
p=json.load(open("/tmp/agent-skills-manager-selftest-plan.json"))
assert p["version"]==1
assert "summary" in p
print("PASS  plan json schema")
PY

echo
echo "passed=$pass failed=$fail"
echo "NOTE: apply/undo against real ~/.agents NOT run (destructive)."
if [[ "$fail" -ne 0 ]]; then
  exit 1
fi
