#!/usr/bin/env bash
# Installe krew + le plugin kubectl crust-gather : le framework turtles/test collecte
# alors automatiquement l'état des clusters (management + downstream) en artefacts.
set -euo pipefail

if kubectl crust-gather --version >/dev/null 2>&1; then
  echo "crust-gather déjà installé"
  exit 0
fi

if ! kubectl krew version >/dev/null 2>&1; then
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  OS="$(uname | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"
  KREW="krew-${OS}_${ARCH}"
  curl -fsSLo "$TMP/${KREW}.tar.gz" "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz"
  tar -C "$TMP" -zxf "$TMP/${KREW}.tar.gz"
  "$TMP/${KREW}" install krew
  echo 'Ajouter au PATH : export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"'
fi

export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"
kubectl krew install crust-gather
kubectl crust-gather --version
