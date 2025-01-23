package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strings"

	"example.com/districts"
	"example.com/groups"
	"example.com/members"
	"example.com/messages"
	"example.com/users"
	"github.com/astaxie/beego/session"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	globalSessions *session.Manager
	tlsConfig      *tls.Config
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	globalSessions, err = session.NewManager("file", &session.ManagerConfig{CookieName: os.Getenv("Session_Cookie"), Gclifetime: 3600, ProviderConfig: "./tmp"})
	if err != nil {
		log.Fatalf("Failed to create session manager")
	}
	go globalSessions.GC()
	certificate, err := tls.LoadX509KeyPair("./certificate/cert.pem", "./certificate/key.pem")
	if err != nil {
		log.Fatalf("failed to load server certificates: %v", err)
	}
	tlsConfig = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
	}
}

// All trafic has to go through middleware to check if it is authentic
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", string(os.Getenv("Allow_Origin")))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		if !strings.EqualFold(r.URL.Path, "/login") {
			c, err := r.Cookie(os.Getenv("Session_Cookie"))
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if !globalSessions.GetProvider().SessionExist(c.Value) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)

	})
}

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("Mongo_Connect")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	db := client.Database(os.Getenv("Database"))
	names, err := db.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	u := users.NewUsers(db, globalSessions)
	exists := false
	for _, name := range names {
		if name == "user" {
			exists = true
		}
	}

	if !exists {
		db.CreateCollection(context.TODO(), "user")
		user := users.User{Email: os.Getenv("DefaultEmail"), Password: os.Getenv("DefaultPassword"), Active: true}
		_, err = u.Register(&user)
		if err != nil {
			log.Fatal("Error initializing default users")
		}
	}

	router := http.NewServeMux()
	router.Handle("/login", middleware(http.HandlerFunc(u.ServeHTTP)))
	router.Handle("/logout", middleware(http.HandlerFunc(u.ServeHTTP)))
	router.Handle("/user", middleware(http.HandlerFunc(u.ServeHTTP)))

	m := members.NewMembers(db)
	router.Handle("/member", middleware(http.HandlerFunc(m.ServeHTTP)))

	g := groups.NewGroups(db)
	router.Handle("/group", middleware(http.HandlerFunc(g.ServeHTTP)))

	d := districts.NewDistricts(db)
	router.Handle("/district", middleware(http.HandlerFunc(d.ServeHTTP)))

	mes := messages.NewMessages(db)
	router.Handle("/message", middleware(http.HandlerFunc(mes.ServeHTTP)))

	server := &http.Server{
		Addr:      os.Getenv("PORT"),
		Handler:   router,
		TLSConfig: tlsConfig,
	}
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
