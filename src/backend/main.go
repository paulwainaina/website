package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strings"

	"example.com/users"
	"github.com/astaxie/beego/session"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	config         *tls.Config
	globalSessions *session.Manager
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	cert, err := tls.LoadX509KeyPair("../server.crt", "../server.key")
	if err != nil {
		log.Fatalf("Failed to load X509 key pair: %v", err)
	}
	config = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	globalSessions, err = session.NewManager("file", &session.ManagerConfig{CookieName: os.Getenv("Session_Cookie"), Gclifetime: 3600, ProviderConfig: "./tmp"})
	if err != nil {
		log.Fatalf("Failed to create session manager")
	}
	go globalSessions.GC()
}

// All trafic has to go through middleware to check if it is authentic
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", string(os.Getenv("Allow_Origin")))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				return
			}
			if strings.Compare(r.URL.Path, "/login") != 0 {
				_, err := r.Cookie(os.Getenv("Session_Cookie"))
				if err != nil {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		}
	})
}

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("Mongo_Connect")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	db := client.Database(os.Getenv("Database"))
	_, err = db.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	u := users.NewUsers(nil, globalSessions)
	router := http.NewServeMux()
	router.Handle("/login", middleware(http.HandlerFunc(u.ServeHTTP)))
	router.Handle("/logout", middleware(http.HandlerFunc(u.ServeHTTP)))

	server := &http.Server{
		Addr:      os.Getenv("PORT"),
		Handler:   router,
		TLSConfig: config,
	}
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
