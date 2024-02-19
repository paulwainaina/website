package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/joho/godotenv"
)

var (
	tmp *template.Template
	url string
)

func init() {
	tmp = template.Must(template.ParseGlob("./template/*.html"))
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading the .env file")
	}
	url = fmt.Sprintf("%s://%s:%s", os.Getenv("MODE"), os.Getenv("SERVER"), os.Getenv("PORT"))
}

type Page struct {
	Title      string
	BackendUrl string
}

func RenderTemplate(w http.ResponseWriter, file string, page *Page) {
	err := tmp.ExecuteTemplate(w, file, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	page := &Page{Title: "Home Page", BackendUrl: url}
	RenderTemplate(w, "index.html", page)
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(os.Getenv("ASSETSFOLDER")))))
	http.HandleFunc("/", IndexHandler)
	http.ListenAndServe(fmt.Sprintf("%s:%s", os.Getenv("SERVER"), os.Getenv("PORT")), nil)
}
