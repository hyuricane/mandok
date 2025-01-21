# !/bin/bash
if [[ "$*" == *"--ui"* ]]; then
  ./view.sh
fi
go build -o ./dist/mandok main.go