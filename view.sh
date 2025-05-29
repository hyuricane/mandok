# !/bin/bash
cd nodejs && npx tailwindcss -i ../static/input.css -o ../static/main.css && cd ..
go tool templ generate