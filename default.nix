{ pkgs ? import <nixpkgs> { } }:

# pkgs.buildGoModule rec {
pkgs.buildGoModule {
  pname = "message-to-discord";
  version = "0.1.0";

  src = ./.;

  vendorHash = null;

  ldflags = [
    "-s"
    "-w"
  ];

  postInstall = ''
    mv $out/bin/messageToDiscord $out/bin/message-to-discord
  '';
}
