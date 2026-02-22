.PHONY: milestone-loop

milestone-loop:
	@test -n "$(MILESTONE)" || (echo "MILESTONE is required (example: MILESTONE=M03)" && exit 1)
	@test -n "$(GOAL)" || (echo "GOAL is required" && exit 1)
	@scripts/milestone-review-loop.sh \
		--milestone "$(MILESTONE)" \
		--goal "$(GOAL)" \
		--max-rounds "$${MAX_ROUNDS:-3}" \
		--mode "$${AMP_MODE:-deep}" \
		--visibility "$${AMP_VISIBILITY:-workspace}"
