.PHONY: milestone-worktree milestone-loop

# Usage: make milestone-worktree NUM=04 SLUG=auth-middleware
milestone-worktree:
	@test -n "$(NUM)" || (echo "NUM is required (example: NUM=04)" && exit 1)
	@test -n "$(SLUG)" || (echo "SLUG is required (example: SLUG=auth-middleware)" && exit 1)
	git fetch origin main
	git worktree add ../shiru-m$(NUM) -b milestone/$(NUM)-$(SLUG) origin/main
	@echo "Worktree created at ../shiru-m$(NUM) on branch milestone/$(NUM)-$(SLUG) from origin/main"

milestone-loop:
	@test -n "$(MILESTONE)" || (echo "MILESTONE is required (example: MILESTONE=M03)" && exit 1)
	@test -n "$(GOAL)" || (echo "GOAL is required" && exit 1)
	@scripts/milestone-review-loop.sh \
		--milestone "$(MILESTONE)" \
		--goal "$(GOAL)" \
		--max-rounds "$${MAX_ROUNDS:-3}" \
		--mode "$${AMP_MODE:-deep}" \
		--visibility "$${AMP_VISIBILITY:-workspace}"
