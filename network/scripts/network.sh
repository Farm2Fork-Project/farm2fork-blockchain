#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-}"
case "${ACTION}" in
  up) echo "Starting Farm2Fork Fabric network" ;;
  down) echo "Stopping Farm2Fork Fabric network" ;;
  *) echo "Usage: $0 {up|down}" && exit 1 ;;
esac
