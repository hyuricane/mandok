# !/bin/bash
if [[ "$*" == *"--ui"* ]]; then
  ./view.sh
fi
CGO_ENABLED=0 go build -o ./dist/mandok main.go