package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"inovasiriset.co.id/docker/manager/web"
)

//go:embed all:static
var staticFs embed.FS

func main() {
	staticSubFs, err := fs.Sub(staticFs, "static")
	if err != nil {
		log.Fatalf("[ERROR] failed to load static files: %v", err)
	}
	log.Fatal(web.ListenHttp(map[string][]fs.FS{
		"/static": {os.DirFS("static"), staticSubFs},
	}))
}
