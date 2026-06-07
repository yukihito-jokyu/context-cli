#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

run_in_dev_shell() {
  nix develop --no-update-lock-file --command "$@"
}

validate_tool() {
  case "$1" in
    go)
      run_in_dev_shell go version
      ;;
    gopls)
      run_in_dev_shell gopls version
      ;;
    dlv | delve)
      run_in_dev_shell dlv version
      ;;
    golangci-lint)
      run_in_dev_shell golangci-lint version
      ;;
    task | go-task)
      run_in_dev_shell task --version
      ;;
    nixfmt | nixfmt-rfc-style)
      run_in_dev_shell nixfmt --version
      ;;
    git)
      run_in_dev_shell git --version
      ;;
    lefthook)
      run_in_dev_shell lefthook version
      ;;
    node | nodejs)
      run_in_dev_shell node --version
      ;;
    pnpm)
      run_in_dev_shell pnpm --version
      ;;
    *)
      printf 'Unsupported validation tool: %s\n' "$1" >&2
      printf 'Add a side-effect-free version command to validate_tool first.\n' >&2
      return 2
      ;;
  esac
}

nix fmt --no-update-lock-file -- --check .
nix flake check --no-update-lock-file
nix flake show --all-systems --no-update-lock-file

for tool in go gopls dlv golangci-lint task nixfmt git lefthook node pnpm "$@"; do
  validate_tool "$tool"
done
