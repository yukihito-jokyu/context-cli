{
  description = "Development environment for context-cli";

  nixConfig = {
    extra-experimental-features = [
      "nix-command"
      "flakes"
    ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        # TODO: Add packages.default after the CLI entry point and build layout are decided.
        devShells.default = pkgs.mkShellNoCC {
          packages = with pkgs; [
            actionlint
            delve
            git
            go
            go-task
            golangci-lint
            gotestsum
            govulncheck
            gopls
            lefthook
            nixfmt
            nodejs
            pnpm
          ];

          shellHook = ''
            workspace_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
            export WORKSPACE_ROOT="$workspace_root"
            export TOOLCHAIN_ROOT="$WORKSPACE_ROOT/.toolchain"
            export GOCACHE="$TOOLCHAIN_ROOT/go/cache"
            export GOENV="$TOOLCHAIN_ROOT/go/env"
            export GOPATH="$TOOLCHAIN_ROOT/go/path"
            export GOMODCACHE="$GOPATH/pkg/mod"
            export PATH="$PATH:$GOPATH/bin"
            export CGO_ENABLED=0

            mkdir -p \
              "$GOCACHE" \
              "$GOPATH/bin" \
              "$GOMODCACHE"

            if git rev-parse --git-dir >/dev/null 2>&1; then
              lefthook install
            fi
          '';
        };

        formatter = pkgs.nixfmt;
      }
    );
}
