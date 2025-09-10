# !/bin/bash
go tool templ generate
cd nodejs && npx tailwindcss -i ../static/input.css -o ../static/main.css && cd ..