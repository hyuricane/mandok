#!/bin/sh
set -e

usage() {
  echo "Usage: $0 [options]"
  echo "Options:"
  echo "  --ui    Run UI"
  echo "  --help  Show this help message"
  exit 1
}

while [ $# -gt 0 ]; do
  case $1 in
    --ui)
      ./view.sh
      shift
      ;;
    --help|-h)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
done

CGO_ENABLED=0 go build -tags prod -trimpath -ldflags "-s -w" -o ./dist/mandok main.go
