package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/astaxie/beego/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const userCollection = "user"

type User struct {
	Name     string `bson:"Name"`
	Email    string `bson:"Email"`
	Password string `bson:"Password"`
	Active   bool   `bson:"Active"`
	Premium  bool   `bson:"Premium"`
	Passport string `bson:"Passport"`
	Role     int    `bson:"Role"`
}

type Users struct {
	users          []*User
	globalSessions *session.Manager
	db             *mongo.Database
}

func NewUsers(db *mongo.Database, globalSessions *session.Manager) *Users {
	users := make([]*User, 0)
	col := db.Collection(userCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err.Error())
	} else {
		if err = result.All(context.TODO(), &users); err != nil {
			log.Fatal("Error loading users data " + err.Error())
		}
	}
	return &Users{users: users, globalSessions: globalSessions, db: db}
}

func (users *Users) login(username, userpassword string) (*User, error) {
	for _, user := range users.users {
		if strings.EqualFold(username, user.Name) || strings.EqualFold(username, user.Email) {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userpassword))
			if err != nil {
				return nil, fmt.Errorf("Wrong password provided")
			}
			return user, nil
		}
	}
	return nil, fmt.Errorf("Account does not exist")
}

func (users *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/login") {
		var credentials struct {
			username     string
			userpassword string
		}
		err := json.NewDecoder(r.Body).Decode(&credentials)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user, err := users.login(credentials.username, credentials.userpassword)
		if err != nil {
			res := struct{ Error string }{Error: err.Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
		sess, err := users.globalSessions.SessionStart(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer sess.SessionRelease(w)
		sess.Set("useremail", user.Email)
		//http.SetCookie(w, &http.Cookie{Name: os.Getenv("Session_Cookie"), Value: sess.SessionID(), HttpOnly: true, Secure: true})
		json.NewEncoder(w).Encode(user)
		return
	} else if strings.EqualFold(r.URL.Path, "/logout") {
		users.globalSessions.SessionDestroy(w, r)
	}

}
