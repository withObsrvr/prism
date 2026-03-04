{
  description = "Prism - Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go;

        devTools = with pkgs; [
          # Go development
          go
          gopls
          gotools
          go-tools
          delve

          # Build tools
          gnumake
          gcc
          pkg-config

          # Templ + Tailwind
          templ
          nodejs_20

          # Development utilities
          air
          curl
          jq
          git

          # Nix tools
          nixpkgs-fmt
          nil
        ];

        shellHook = ''
          echo "Prism development environment"
          echo ""
          echo "Commands:"
          echo "  make dev      - Start dev server with hot reload"
          echo "  make build    - Build the binary"
          echo "  make test     - Run tests"
          echo "  make generate - Generate templ files"
          echo "  make css      - Build Tailwind CSS"
          echo ""
          echo "Go version: $(go version)"
          echo ""

          # Go environment
          export GOPATH="$PWD/.go"
          export GOCACHE="$PWD/.go/cache"
          export GOMODCACHE="$PWD/.go/mod"
          export PATH="$GOPATH/bin:$PWD/node_modules/.bin:$PATH"

          export CGO_ENABLED=0
          export GO111MODULE=on

          mkdir -p .go/{bin,cache,mod}
        '';

      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = devTools;
          inherit shellHook;
        };

        packages.default = pkgs.buildGoModule {
          pname = "prism";
          version = "0.1.0";

          src = ./.;

          vendorHash = null;

          CGO_ENABLED = 0;

          ldflags = [
            "-s" "-w"
            "-X github.com/withObsrvr/prism/cmd/prism.version=${self.shortRev or "dev"}"
          ];

          meta = with pkgs.lib; {
            description = "Prism";
            license = licenses.mit;
          };
        };

        packages.docker = pkgs.dockerTools.buildLayeredImage {
          name = "prism";
          tag = "latest";

          contents = with pkgs; [
            self.packages.${system}.default
            cacert
            tzdata
          ];

          config = {
            Cmd = [ "${self.packages.${system}.default}/bin/prism" "serve" ];
            ExposedPorts = {
              "8080/tcp" = {};
            };
            Env = [
              "PORT=8080"
            ];
          };
        };

        formatter = pkgs.nixpkgs-fmt;
      });
}
