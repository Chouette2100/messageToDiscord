{
  description = "Discord relay for cron/sendmail-style alerts";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  # outputs = { self, nixpkgs }:
  outputs = { nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in {
      packages.${system}.default = import ./default.nix { inherit pkgs; };
    };
}
