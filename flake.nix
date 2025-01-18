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

            # X11 and OpenGL dependencies
            xorg.libX11.dev
            xorg.libXrandr
            xorg.libXinerama
            xorg.libXcursor
            xorg.libXi
            xorg.libXxf86vm
            libGL
            libGLU
            freeglut
            mesa.dev
            xorg.libxcb
            xorg.libXrender
            xorg.libXfixes
            xorg.libXext
            xorg.libXtst
            mesa.drivers

            # Input simulation
            wtype
          ];

          # Set library path for OpenGL
          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [
            pkgs.libGL
            pkgs.libGLU
            pkgs.mesa.drivers
            pkgs.xorg.libX11
          ];

          # Environment variables if needed
          shellHook = ''
            echo "PoE Helper development environment"
            echo "Ready to build with: go build -o poe-helper ./cmd/poe-helper"
          '';
        };
      });
}
