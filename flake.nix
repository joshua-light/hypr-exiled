{
  description = "Hypr Exiled development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            # Go
            go
            gopls
            gotools
            go-tools

            xorg.libX11.dev
            xorg.libXi
            xorg.libxcb
            xorg.libXfixes
            xorg.libXext
            xorg.libXtst

            rofi
          ];

          # Set library path for OpenGL
          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [ pkgs.xorg.libX11 ];

          # Set library path for OpenGL
          shellHook = ''
            echo "Hypr Exiled development environment"
            echo "Ready to build with: go build -o hypr-exiled ./cmd/hypr-exiled"
          '';
        };
      }) // {
        targetSystems = [ "aarch64-linux" "x86_64-linux" ];
      };
}
