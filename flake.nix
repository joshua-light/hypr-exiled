{
  description = "PoE Helper development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go
            go
            gopls
            gotools
            go-tools

            # Build dependencies
            pkg-config

            # Deps
            rofi
          ];

          # Set library path for OpenGL
          shellHook = ''
            echo "PoE Helper development environment"
            echo "Ready to build with: go build -o poe-helper ./cmd/poe-helper"
          '';
        };
      }) // {
        targetSystems = [ "aarch64-linux" "x86_64-linux" ];
      };
}
