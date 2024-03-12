{
  description = "Example Go development environment for Zero to Nix";

  # Flake inputs
  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2305.491812.tar.gz";
  };

  # Flake outputs
  outputs = { self, nixpkgs }:
    let
      # Systems supported
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "x86_64-darwin" # 64-bit Intel macOS
        "aarch64-darwin" # 64-bit ARM macOS
      ];

      # Helper to provide system-specific attributes
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      # Development environment output
      devShells = forAllSystems ({ pkgs }: {
        default = pkgs.mkShell {
          # The Nix packages provided in the environment
          packages = with pkgs; [
            go
            libgit2_1_5
            pkg-config
            gotools # Go tools like goimports, godoc, and others
            gum
            cloc
            gore
          ];
        };
      });

       # Package output
      packages = forAllSystems ({ pkgs }: {
        default = pkgs.buildGoModule {
          pname = "ubik";
          version = "0.1.0";
          vendorSha256 = "sha256-Hk7SECmVO3I7GYBHtPewV/mYx0CRwxFHnW4LNlPrwzU=";

          nativeBuildInputs = with pkgs; [
            pkg-config
          ];

          buildInputs = with pkgs; [
            libgit2_1_5
          ];


          subPackages = [
            "." # Build the package in the current directory
            # Add more packages if needed
          ];

          src = ./.;
        };
      });

      # App output
      apps = forAllSystems ({ pkgs }: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.system}.default}/bin/ubik";
        };
      });
    };
}
