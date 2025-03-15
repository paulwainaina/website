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
	certificate, err := tls.LoadX509KeyPair("./certificate/cert.pem", "./certificate/key.pem")
	if err != nil {
		log.Fatalf("failed to load server certificates: %v", err)
	}
	tlsConfig = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
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

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "profile.html", &Page{Title: "Profile", Data: nil})
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

func MembersHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "members.html", &Page{Title: "Members", Data: nil})
}

func GroupsHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "groups.html", &Page{Title: "Groups", Data: nil})
}

func DistrictsHandler(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "districts.html", &Page{Title: "Districts", Data: nil})
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/home", IndexHandler)
	router.Handle("/services", middleware(http.HandlerFunc(ServicesHandler)))
	router.HandleFunc("/profile", ProfileHandler)
	router.HandleFunc("/contacts", ContactsHandler)
	router.HandleFunc("/login", LoginHandler)
	router.HandleFunc("/signup", SignupHandler)
	router.HandleFunc("/members", MembersHandler)
	router.HandleFunc("/groups", GroupsHandler)
	router.HandleFunc("/districts", DistrictsHandler)
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
