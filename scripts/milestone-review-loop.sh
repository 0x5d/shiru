#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/milestone-review-loop.sh --milestone <id> --goal <text> [--max-rounds <n>] [--mode <mode>] [--visibility <visibility>]

Example:
  scripts/milestone-review-loop.sh \
    --milestone M03 \
    --goal "Implement Google login and per-user data isolation per checklist" \
    --max-rounds 3 \
    --mode deep
EOF
}

MILESTONE=""
GOAL=""
MAX_ROUNDS=3
MODE="deep"
VISIBILITY="workspace"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --milestone)
      MILESTONE="${2:-}"
      shift 2
      ;;
    --goal)
      GOAL="${2:-}"
      shift 2
      ;;
    --max-rounds)
      MAX_ROUNDS="${2:-}"
      shift 2
      ;;
    --mode)
      MODE="${2:-}"
      shift 2
      ;;
    --visibility)
      VISIBILITY="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$MILESTONE" || -z "$GOAL" ]]; then
  echo "Both --milestone and --goal are required." >&2
  usage
  exit 1
fi

if ! [[ "$MAX_ROUNDS" =~ ^[0-9]+$ ]] || [[ "$MAX_ROUNDS" -lt 1 ]]; then
  echo "--max-rounds must be a positive integer." >&2
  exit 1
fi

if ! command -v amp >/dev/null 2>&1; then
  echo "amp CLI not found in PATH." >&2
  exit 1
fi

branch_name="$(git branch --show-current)"
worktree_path="$(pwd)"
if [[ "$branch_name" != *"$MILESTONE"* && "$worktree_path" != *"$MILESTONE"* ]]; then
  echo "Current branch/worktree does not appear to match milestone '$MILESTONE'." >&2
  echo "Branch: $branch_name" >&2
  echo "Path:   $worktree_path" >&2
  echo "Switch to the milestone worktree before running this script." >&2
  exit 1
fi

echo "Creating implementer thread..."
implementer_thread="$(amp threads new --visibility "$VISIBILITY" | tr -d '[:space:]')"

implementer_prompt=$(cat <<EOF
Milestone: $MILESTONE
Goal: $GOAL

Execution requirements:
- Work directly in this workspace and implement the milestone fully.
- Follow AGENTS.md instructions exactly.
- Run tests/lint relevant to touched code.
- End your response with:
  STATUS: READY_FOR_REVIEW
  SUMMARY: <short summary>
EOF
)

echo "Running implementer pass in thread $implementer_thread"
amp --dangerously-allow-all -m "$MODE" \
  threads continue "$implementer_thread" \
  -x "$implementer_prompt" >/tmp/amp-impl-${MILESTONE}.log

echo "Implementer pass complete. Starting adversarial review loop (max rounds: $MAX_ROUNDS)."

approved=0
for round in $(seq 1 "$MAX_ROUNDS"); do
  echo "Round $round: creating reviewer thread..."
  reviewer_thread="$(amp threads new --visibility "$VISIBILITY" | tr -d '[:space:]')"

  reviewer_prompt=$(cat <<EOF
You are the adversarial reviewer for milestone $MILESTONE.

Context:
- Implementer thread ID: $implementer_thread
- Goal: $GOAL

Your job:
- Review current workspace changes with a hostile mindset.
- Find regressions, security issues, tenant-isolation leaks, migration risks, and missing tests.
- Prioritize concrete evidence with file and line references.
- Require explicit tests for every high/medium issue.

Output format (strict):
VERDICT: APPROVED|REJECTED
HIGH: <count>
MEDIUM: <count>
LOW: <count>
FINDINGS:
- [severity] file:line - issue, risk, required fix
EOF
)

  review_output="$(amp --dangerously-allow-all -m "$MODE" threads continue "$reviewer_thread" -x "$reviewer_prompt")"
  printf '%s\n' "$review_output" >"/tmp/amp-review-${MILESTONE}-round-${round}.log"

  echo "Reviewer thread: $reviewer_thread"
  if printf '%s\n' "$review_output" | grep -q "^VERDICT: APPROVED"; then
    approved=1
    echo "Milestone approved in round $round."
    break
  fi

  echo "Milestone rejected in round $round. Sending findings back to implementer thread."
  fix_prompt=$(cat <<EOF
Reviewer rejected milestone $MILESTONE in round $round.

Apply all fixes from this review output, then run relevant checks and tests.
When done, end your response with:
STATUS: READY_FOR_REVIEW
SUMMARY: <short summary>

Reviewer output:
$review_output
EOF
)

  amp --dangerously-allow-all -m "$MODE" \
    threads continue "$implementer_thread" \
    -x "$fix_prompt" >/tmp/amp-impl-${MILESTONE}-round-${round}.log
done

echo
echo "Implementer thread: $implementer_thread"
echo "Implementer URL: https://ampcode.com/threads/$implementer_thread"

if [[ "$approved" -eq 1 ]]; then
  echo "Final status: APPROVED"
  exit 0
fi

echo "Final status: REJECTED_AFTER_${MAX_ROUNDS}_ROUNDS"
exit 2
