{ pkgs }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
    golangci-lint
    gotools
    sqlite
  ];

  shellHook = ''
    echo "shu development shell"
    echo "Go $(go version | awk '{print $3}')"
  '';
}
