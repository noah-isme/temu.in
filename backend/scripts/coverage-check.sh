#!/usr/bin/env bash
# coverage-check.sh — per-package statement coverage gate.
#
# Usage: TEST_DATABASE_URL=... ./scripts/coverage-check.sh coverage.out
#
# Reads a single `go test -coverprofile` file (no second test run), groups the
# raw block lines `<file>:<start>,<end> <numStmts> <count>` by package
# directory and computes sum(numStmts where count>0) / sum(numStmts).
#
# Gate:
#   - booking, payment, availability, auth must be >= 60%
#   - token, user, review, favorite, middleware, admin, audit, service must be > 0%
#
# The thresholds assume the integration suite ran (CI with Postgres and
# REQUIRE_INTEGRATION=1). The script therefore refuses to run when
# TEST_DATABASE_URL is unset, so a unit-only profile is never mistaken for a
# passing (or failing) gate.
set -euo pipefail

profile="${1:-}"
if [[ -z "$profile" ]]; then
  echo "usage: $0 <coverage.out>" >&2
  exit 2
fi
if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  echo "coverage-check: TEST_DATABASE_URL is unset; the gate is only meaningful for a profile produced with the integration suite (REQUIRE_INTEGRATION=1)" >&2
  exit 2
fi
if [[ ! -s "$profile" ]]; then
  echo "coverage-check: profile '$profile' is missing or empty" >&2
  exit 2
fi

module_prefix="temu.in/backend/internal/"
min_pkgs="booking payment availability auth"
nonzero_pkgs="token user review favorite middleware admin audit service"
min_pct=60

awk -v prefix="$module_prefix" -v min_pkgs="$min_pkgs" -v nonzero_pkgs="$nonzero_pkgs" -v min_pct="$min_pct" '
NR == 1 && /^mode:/ { next }
NF == 3 {
  block = $1
  # A block can repeat when several test binaries cover the same package;
  # count each block once and treat it as covered if any run covered it.
  if (!(block in stmts)) {
    stmts[block] = $2
    covered[block] = 0
  }
  if ($3 + 0 > 0) covered[block] = 1
}
END {
  for (block in stmts) {
    file = block
    sub(/:.*/, "", file)
    pkg = file
    sub(/\/[^\/]*$/, "", pkg)
    total[pkg] += stmts[block]
    if (covered[block]) hit[pkg] += stmts[block]
  }

  printf "%-50s %8s %8s %8s\n", "package", "stmts", "covered", "percent"
  n = 0
  for (pkg in total) names[++n] = pkg
  # insertion sort for stable, readable output (no gawk asort dependency)
  for (i = 2; i <= n; i++) {
    v = names[i]; j = i - 1
    while (j > 0 && names[j] > v) { names[j + 1] = names[j]; j-- }
    names[j + 1] = v
  }
  for (i = 1; i <= n; i++) {
    pkg = names[i]
    pct = total[pkg] ? 100 * hit[pkg] / total[pkg] : 0
    printf "%-50s %8d %8d %7.1f%%\n", pkg, total[pkg], hit[pkg], pct
  }

  failed = 0
  split(min_pkgs, mins, " ")
  for (k in mins) {
    pkg = prefix mins[k]
    pct = total[pkg] ? 100 * hit[pkg] / total[pkg] : 0
    if (!(pkg in total)) { printf "FAIL %s: not in profile\n", pkg; failed = 1 }
    else if (pct < min_pct) { printf "FAIL %s: %.1f%% < %d%%\n", pkg, pct, min_pct; failed = 1 }
  }
  split(nonzero_pkgs, nz, " ")
  for (k in nz) {
    pkg = prefix nz[k]
    if (!(pkg in total) || hit[pkg] == 0) { printf "FAIL %s: 0%% coverage\n", pkg; failed = 1 }
  }
  if (failed) exit 1
  print "coverage-check: OK"
}
' "$profile"
