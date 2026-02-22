## Milestone Review Loop Automation

This repo includes a script to run an implementer + adversarial reviewer loop for a single milestone.

### Prerequisites

- `amp` CLI installed and logged in
- Current branch/worktree name includes the milestone ID (for example `M03`)

### One-command usage

```bash
make milestone-loop MILESTONE=M03 GOAL="Implement Google login and per-user data isolation"
```

Optional environment variables:

- `MAX_ROUNDS` (default `3`)
- `AMP_MODE` (default `deep`)
- `AMP_VISIBILITY` (default `workspace`)

Example:

```bash
MAX_ROUNDS=5 AMP_MODE=smart make milestone-loop \
  MILESTONE=M03 \
  GOAL="Implement Google login and per-user data isolation"
```

### What it does

1. Creates an implementer thread and runs implementation prompt.
2. Creates a fresh adversarial reviewer thread for each round.
3. If reviewer rejects, sends findings to implementer thread for fixes.
4. Repeats until approval or max rounds reached.

The script prints implementer thread ID and URL at the end.

### Direct script usage

```bash
scripts/milestone-review-loop.sh \
  --milestone M03 \
  --goal "Implement Google login and per-user data isolation" \
  --max-rounds 3 \
  --mode deep \
  --visibility workspace
```
