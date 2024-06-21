package main

import (
	"log"

	"github.com/joho/godotenv"
	"inovasiriset.co.id/docker/manager/web"
)

func main() {
	godotenv.Load()

	log.Fatal(web.ListenHttp())
}
