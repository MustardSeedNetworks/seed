#!/bin/bash
set -e

echo "Running security checks..."

# Check .env is not staged
if git diff --cached --name-only | grep -E '^\.env$|^\.env\.' | grep -v '\.example$'; then
  echo "ERROR: .env file is staged for commit!"
  exit 1
fi

# Check for common secret patterns
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)
if [ -n "$STAGED_FILES" ]; then
  if echo "$STAGED_FILES" | xargs grep -l -E '(password|secret|api_key|apikey|token)\s*[:=]\s*["\047][^"\047]{8,}["\047]' 2>/dev/null; then
    echo "WARNING: Possible secrets found in staged files - please verify"
  fi
fi

# Check for VITE_* secrets (common mistake)
if [ -n "$STAGED_FILES" ]; then
  if echo "$STAGED_FILES" | xargs grep -l -E 'VITE_.*SECRET|VITE_.*KEY.*=.*[a-zA-Z0-9]{20,}' 2>/dev/null; then
    echo "ERROR: Secrets in VITE_* env vars are exposed to client!"
    exit 1
  fi
fi

# Dependency audit - npm
if [ -f "package.json" ]; then
  echo "Checking npm dependencies..."
  npm audit --audit-level=high 2>/dev/null || echo "Warning: npm audit found issues"
fi

# Dependency audit - Go
if [ -f "go.mod" ]; then
  echo "Checking Go dependencies..."
  if command -v govulncheck &> /dev/null; then
    govulncheck ./... 2>/dev/null || echo "Warning: govulncheck found issues"
  else
    echo "⚠ govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"
  fi
fi

echo "Security checks complete!"
