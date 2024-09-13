{
  description = "Example Go development environment for Zero to Nix";

  # Flake inputs
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
  };

  # Flake outputs
  outputs = {
    self,
    nixpkgs,
  }: let
    # Systems supported
    allSystems = [
      "x86_64-linux" # 64-bit Intel/AMD Linux
      "aarch64-linux" # 64-bit ARM Linux
      "x86_64-darwin" # 64-bit Intel macOS
      "aarch64-darwin" # 64-bit ARM macOS
    ];

    # Helper to provide system-specific attributes
    forAllSystems = f:
      nixpkgs.lib.genAttrs allSystems (system:
        f {
          pkgs = import nixpkgs {inherit system;};
        });
  in {
    # Development environment output
    devShells = forAllSystems ({pkgs}: {
      default = pkgs.mkShell {
        # The Nix packages provided in the environment
        packages = with pkgs; [
          go_1_23
          gopls
          gosec
        ];
      };
    });

    # Package output
    packages = forAllSystems ({pkgs}: {
      default = pkgs.buildGoModule.override {go = pkgs.go_1_23;} {
        pname = "ubik";
        version = "pre-alpha";
        vendorHash = "sha256-Wj0XPigjXWRf4/8Y/EdQ/1P1HypghiP0402ZegUxYvk=";

        buildInputs = with pkgs; [
          git
        ];

        subPackages = [
          "." # Build the package in the current directory
          # Add more packages if needed
        ];

        src = ./.;
      };
    });
  };
}
