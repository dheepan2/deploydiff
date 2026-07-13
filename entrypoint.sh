#!/bin/sh
set -eu

before=$1
after=$2
output=$3

if [ -n "${GITHUB_WORKSPACE:-}" ]; then
  cd "$GITHUB_WORKSPACE"
fi

report=$(deploydiff --output "$output" compare "$before" "$after")
printf '%s\n' "$report"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "report<<DEPLOYDIFF_REPORT"
    printf '%s\n' "$report"
    echo "DEPLOYDIFF_REPORT"
  } >> "$GITHUB_OUTPUT"
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "## DeployDiff"
    echo
    echo '```text'
    printf '%s\n' "$report"
    echo '```'
  } >> "$GITHUB_STEP_SUMMARY"
fi
