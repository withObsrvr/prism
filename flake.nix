{
  description = "Prism - Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config = {
            allowUnfree = true;
          };
        };
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

          # Playwright for E2E testing
          playwright-driver
        ];

        # Playwright runtime dependencies
        playwrightRuntimeDeps = with pkgs; [
          glib
          nss
          nspr
          dbus
          atk
          at-spi2-core
          at-spi2-atk
          cups
          libdrm
          expat
          libxkbcommon
          xorg.libxcb
          xorg.libX11
          xorg.libXcomposite
          xorg.libXdamage
          xorg.libXext
          xorg.libXfixes
          xorg.libXrandr
          mesa
          libgbm
          pango
          cairo
          alsa-lib
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

          # Playwright runtime dependencies
          export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
          export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath playwrightRuntimeDeps}:$LD_LIBRARY_PATH"

          export PS1='\[\033[1;35m\][prism]\[\033[0m\] \[\033[1;32m\]\u@\h\[\033[0m\]:\[\033[1;36m\]\w\[\033[0m\]\$ '
        '';

      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = devTools ++ playwrightRuntimeDeps;
          inherit shellHook;
        };

        packages.default = pkgs.buildGoModule {
          pname = "prism";
          version = "0.1.0";

          src = ./.;

          vendorHash = null;

          env.CGO_ENABLED = 0;

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
