#!/bin/bash

set -e

# Install Go tools (golangci-lint, deadcode) tracked in go.mod.
if ! go tool golangci-lint version &> /dev/null || ! go tool deadcode -help &> /dev/null; then
	echo "Installing Go tools..."
	go install tool
else
	echo "Go tools already installed."
fi

# Install non-Go tools.
if command -v semgrep &> /dev/null; then
	echo "semgrep already installed."
else
	echo "Installing semgrep..."
	if command -v pip3 &> /dev/null; then
		pip3 install semgrep
	elif command -v brew &> /dev/null; then
		brew install semgrep
	else
		echo "Error: neither pip3 nor brew found. Install semgrep manually: https://semgrep.dev/docs/getting-started/"
		exit 1
	fi
fi

echo "All tools installed."
