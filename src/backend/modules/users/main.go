package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/astaxie/beego/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const userCollection = "user"

type User struct {
	Id       string `bson:"Id"`
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
			log.Fatal("error loading users data " + err.Error())
		}
	}
	return &Users{users: users, globalSessions: globalSessions, db: db}
}

func (users *Users) login(username, userpassword string) (*User, error) {
	for _, user := range users.users {
		if strings.EqualFold(username, user.Name) || strings.EqualFold(username, user.Email) {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userpassword))
			if err != nil {
				return nil, fmt.Errorf("wrong password provided")
			}
			return user, nil
		}
	}
	return nil, fmt.Errorf("account does not exist")
}

func (users *Users) register(usr *User) (*User, error) {
	for _, user := range users.users {
		if strings.EqualFold(user.Email, usr.Email) {
			return nil, fmt.Errorf("user account already exists")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(usr.Password), 10)
	if err != nil {
		return nil, fmt.Errorf("error processing user password")
	}
	usr.Password = string(hash)
	bsonData, err := bson.Marshal(usr)
	if err != nil {
		return nil, fmt.Errorf("error processing user details")
	}
	col := users.db.Collection(userCollection)
	_, err = col.InsertOne(context.TODO(), bsonData)
	if err != nil {
		return nil, fmt.Errorf("error registering user")
	}
	go func() {
		users.users = append(users.users, usr)
	}()
	return usr, nil
}

func (users *Users) delete(usr *User) (*User, error) {
	for x, user := range users.users {
		if strings.EqualFold(user.Email, usr.Email) && strings.EqualFold(usr.Id, user.Id) {
			bsonData, err := bson.Marshal(usr)
			if err != nil {
				return nil, fmt.Errorf("error processing user details")
			}
			col := users.db.Collection(userCollection)
			_, err = col.DeleteOne(context.TODO(), bsonData)
			if err != nil {
				return nil, fmt.Errorf("error deleting user")
			}
			go func() {
				users.users = append(users.users[:x], users.users[x+1:]...)
			}()
			return user, nil
		}
	}
	return nil, fmt.Errorf("user account does not exists")
}

func (users *Users) update(update map[string]interface{}) (*User, error) {
	var usr User
	var fields []string
	rv := reflect.ValueOf(usr)
	for key, value := range update {
		field := rv.FieldByName(key)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(value)
			if field.Type() == val.Type() {
				field.Set(val)
			} else {
				return nil, fmt.Errorf("update field with type mismatch %v ", key)
			}
			fields = append(fields, key)
		}
	}
	req := 0
	for i, v := range fields {
		if strings.EqualFold(v, "Id") || strings.EqualFold(v, "Email") {
			req += 1
		}
		if i == len(fields)-1 && req != 2 {
			return nil, fmt.Errorf("update missing important fields")
		}
	}
	for _, user := range users.users {
		if strings.EqualFold(user.Email, usr.Email) && strings.EqualFold(user.Id, usr.Id) {
			if usr.Password != "" {
				hash, err := bcrypt.GenerateFromPassword([]byte(usr.Password), 10)
				if err != nil {
					return nil, fmt.Errorf("error processing user password")
				}
				usr.Password = string(hash)
			}
			col := users.db.Collection(userCollection)
			_, err := col.UpdateByID(context.TODO(), usr.Id, usr)
			if err != nil {
				return nil, fmt.Errorf("error updating user")
			}
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("user account does not exists")
}

func (users *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/login") {
		var credentials struct {
			NameEmail string
			Password  string
		}
		err := json.NewDecoder(r.Body).Decode(&credentials)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user, err := users.login(credentials.NameEmail, credentials.Password)
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
		json.NewEncoder(w).Encode(user)
		return
	} else if strings.EqualFold(r.URL.Path, "/logout") {
		users.globalSessions.SessionDestroy(w, r)
	} else if strings.EqualFold(r.URL.Path, "/user") {
		switch r.Method {
		case http.MethodPost:
			{
				var newUser User
				err := json.NewDecoder(r.Body).Decode(&newUser)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := users.register(&newUser)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodGet:
			{
				if len(users.users) == 0 {
					json.NewEncoder(w).Encode("{error: no users found}")
					return
				}
				json.NewEncoder(w).Encode(fmt.Sprintf("{users: %v}", users.users))
				return
			}
		case http.MethodDelete:
			{
				var newUser User
				err := json.NewDecoder(r.Body).Decode(&newUser)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := users.delete(&newUser)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodPut:
			{
				updateUser := make(map[string]interface{}, 0)
				err := json.NewDecoder(r.Body).Decode(&updateUser)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := users.update(updateUser)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return

			}

		}

	}

}
