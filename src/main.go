package main

import (
	"crypto/tls"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var(
	tpl *template.Template
	config *tls.Config
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load X509 key pair: %v", err)
	}
	config = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

type Page struct {
	Title string
	Data  interface{}
}

func RenderTemplate(w http.ResponseWriter, file string, page *Page) {
	err := tpl.ExecuteTemplate(w, file, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func IndexHandler(w http.ResponseWriter,r *http.Request){
	RenderTemplate(w,"index.html",&Page{Title: "Home",Data:nil})
}

func main() {
	router:=http.NewServeMux()
	router.HandleFunc("/",IndexHandler)
	server := &http.Server{
		Addr:      os.Getenv("PORT"),
		Handler:   router,
		TLSConfig: config,
	}
	err:=server.ListenAndServeTLS("","")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}