package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/joho/godotenv"
	"inovasiriset.co.id/docker/manager/web"
)

//go:embed static/*
var staticFs embed.FS

func init() {
	if err := godotenv.Load(); err == nil {
		log.Printf("[INFO] .env file loaded")
	}
}

func main() {
	staticSubFs, err := fs.Sub(staticFs, "static")
	if err != nil {
		log.Fatalf("[ERROR] failed to load static files: %v", err)
	}
	log.Fatal(web.ListenHttp(map[string][]fs.FS{
		"/static": {os.DirFS("static"), staticSubFs},
	}))
}
