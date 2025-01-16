package main

import (
	"crypto/tls"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var (
	tpl       *template.Template
	tlsConfig *tls.Config
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
	certificate, err := tls.LoadX509KeyPair("../certificate/cert.pem", "../certificate/key.pem")
	if err != nil {
		log.Fatalf("failed to load server certificates: %v", err)
	}
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{certificate},
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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "index.html", &Page{Title: "Home", Data: nil})
}

func ServicesHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "services.html", &Page{Title: "Services", Data: nil})
}

func EventsHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "events.html", &Page{Title: "Events", Data: nil})
}

func ContactsHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "contacts.html", &Page{Title: "Contacts", Data: nil})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "login.html", &Page{Title: "Login", Data: nil})
}
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "register.html", &Page{Title: "Login", Data: nil})
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/home", IndexHandler)
	router.HandleFunc("/services", ServicesHandler)
	router.HandleFunc("/events", EventsHandler)
	router.HandleFunc("/contacts", ContactsHandler)
	router.HandleFunc("/login", LoginHandler)
	router.HandleFunc("/signup", SignupHandler)
	router.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	//Redirect unknown path to home
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/home", http.StatusFound)
	})

	server := &http.Server{
		Addr:      os.Getenv("PORT"),
		Handler:   router,
		TLSConfig: tlsConfig,
	}
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
