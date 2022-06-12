package main

import (
	"net/http"
	"os"

	"github.com/lmousom/jisce-soochana/controllers"
)

func main() {
	port := os.Getenv("PORT")
	controllers.NoticeController()
	r := controllers.NoticeRouter()
	http.ListenAndServe(":"+port, r)
}