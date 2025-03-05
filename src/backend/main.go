package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"example.com/districts"
	"example.com/groups"
	"example.com/members"
	"example.com/messages"
	"github.com/astaxie/beego/session"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	globalSessions *session.Manager
	tlsConfig      *tls.Config
	db             *mongo.Database
	m              *members.Members
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
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
			cokval := "000"
			if err == nil {
				cokval = c.Value
			}
			if strings.EqualFold(r.URL.Path, "/loggedin") {
				store, err := globalSessions.GetProvider().SessionRead(cokval)
				if err != nil {
					res := struct{ Error string }{Error: err.Error()}
					json.NewEncoder(w).Encode(res)
					return
				}
				x, _ := store.Get("useremail").(string)
				res := fmt.Sprintf(`{"active": %t}`, m.SuperUser(x))
				json.NewEncoder(w).Encode(res)
				return
			}
			if !globalSessions.GetProvider().SessionExist(cokval) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)

	})
}
func EmptyHandler(w http.ResponseWriter, r *http.Request) {
}

func main() {
	var err error
	globalSessions, err = session.NewManager("file", &session.ManagerConfig{CookieName: os.Getenv("Session_Cookie"), Gclifetime: 3600, ProviderConfig: "./tmp"})
	if err != nil {
		log.Fatalf("Failed to create session manager")
	}
	go globalSessions.GC()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("Mongo_Connect")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())
	db = client.Database(os.Getenv("Database"))
	m = members.NewMembers(db, globalSessions)
	names, err := db.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	exists := false
	for _, name := range names {
		if name == "member" {
			exists = true
		}
	}
	if !exists {
		db.CreateCollection(context.TODO(), "member")
		member := members.Member{Email: os.Getenv("DefaultEmail"), Password: os.Getenv("DefaultPassword"), Active: true, Id: uuid.NewString(), Role: 1}
		_, err = m.Add(&member)
		if err != nil {
			log.Fatal("Error initializing default user")
		}
	}

	certificate, err := tls.LoadX509KeyPair("./certificate/cert.pem", "./certificate/key.pem")
	if err != nil {
		log.Fatalf("failed to load server certificates: %v", err)
	}
	tlsConfig = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
	}

	router := http.NewServeMux()

	router.Handle("/member", middleware(http.HandlerFunc(m.ServeHTTP)))
	router.Handle("/login", middleware(http.HandlerFunc(m.ServeHTTP)))
	router.Handle("/logout", middleware(http.HandlerFunc(m.ServeHTTP)))

	g := groups.NewGroups(db)
	router.Handle("/group", middleware(http.HandlerFunc(g.ServeHTTP)))

	d := districts.NewDistricts(db)
	router.Handle("/district", middleware(http.HandlerFunc(d.ServeHTTP)))

	mes := messages.NewMessages(db)
	router.Handle("/message", middleware(http.HandlerFunc(mes.ServeHTTP)))

	router.Handle("/loggedin", middleware(http.HandlerFunc(EmptyHandler)))

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
