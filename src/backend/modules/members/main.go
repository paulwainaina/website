package members

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/astaxie/beego/session"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type Member struct {
	Id              string `bson:"Id"`
	Name            string `bson:"Name"`
	Email           string `bson:"Email"`
	Contacts        string `bson:"Contacts"`
	DateofBirth     string `bson:"DateofBirth"`
	DateofBaptism   string `bson:"DateofBaptism"`
	DateofCatechism string `bson:"DateofCatechism"`
	District        string `bson:"District"`
	Groups          string `bson:"Groups"`
	Passport        string `bson:"Passport"`
	Password        string `bson:"Password"`
	Active          bool   `bson:"Active"`
	Role            int    `bson:"Role"`
}

type Members struct {
	members        []*Member
	db             *mongo.Database
	globalSessions *session.Manager
}

const memberCollection = "member"

func NewMembers(db *mongo.Database, globalSessions *session.Manager) *Members {
	members := make([]*Member, 0)
	col := db.Collection(memberCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err.Error())
	} else {
		if err = result.All(context.TODO(), &members); err != nil {
			log.Fatal("error loading members data " + err.Error())
		}
	}
	return &Members{members: members, db: db, globalSessions: globalSessions}
}

func (members *Members) login(username, userpassword string) (*Member, error) {
	for _, user := range members.members {
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

func (members *Members) SuperUser(useremail string) bool {
	canedit := false
	for _, user := range members.members {
		if strings.EqualFold(useremail, user.Email) {
			if user.Active {
				canedit = true
			}
			break
		}
	}
	return canedit
}

func (members *Members) Add(newmember *Member) (*Member, error) {
	for _, member := range members.members {
		if strings.EqualFold(member.Email, newmember.Email) {
			return nil, fmt.Errorf("member already exists")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newmember.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error processing user password")
	}
	newmember.Password = string(hash)
	bsonData, err := bson.Marshal(newmember)
	if err != nil {
		return nil, fmt.Errorf("error processing member details")
	}
	col := members.db.Collection(memberCollection)
	_, err = col.InsertOne(context.TODO(), bsonData)
	if err != nil {
		return nil, fmt.Errorf("error registering user")
	}
	go func() {
		members.members = append(members.members, newmember)
	}()
	return newmember, nil
}

func (members *Members) delete(oldmember *Member) (*Member, error) {
	for x, member := range members.members {
		if strings.EqualFold(member.Email, oldmember.Email) && strings.EqualFold(member.Id, oldmember.Id) {
			bsonData, err := bson.Marshal(oldmember)
			if err != nil {
				return nil, fmt.Errorf("error processing member details")
			}
			col := members.db.Collection(memberCollection)
			_, err = col.DeleteOne(context.TODO(), bsonData)
			if err != nil {
				return nil, fmt.Errorf("error deleting user")
			}
			go func() {
				members.members = append(members.members[:x], members.members[x+1:]...)
			}()
			return oldmember, nil
		}
	}
	return nil, fmt.Errorf("member account does not exists")
}

func (members *Members) update(update map[string]interface{}) (*Member, error) {
	usr := Member{}
	for key, value := range update {
		field := reflect.ValueOf(&usr).Elem().FieldByName(key)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(value)
			if field.Type() == val.Type() {
				field.Set(val)
			} else {
				return nil, fmt.Errorf("type mismatch for field %s", key)
			}
		}
	}
	for _, member := range members.members {
		if strings.EqualFold(member.Email, usr.Email) && strings.EqualFold(member.Id, usr.Id) {
			update := bson.M{}
			bsonData, err := bson.Marshal(usr)
			if err != nil {
				return nil, fmt.Errorf("error updating member %s", err)
			}
			bson.Unmarshal(bsonData, &update)
			col := members.db.Collection(memberCollection)
			_, err = col.UpdateOne(context.TODO(), bson.M{"Id": usr.Id}, bson.M{"$set": update})
			if err != nil {
				return nil, fmt.Errorf("error updating member %s", err)
			}
			*member = usr
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("member account does not exists")
}

func (members *Members) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		user, err := members.login(credentials.NameEmail, credentials.Password)
		if err != nil {
			res := struct{ Error string }{Error: err.Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
		if user.Role == 1 && !user.Active {
			res := struct{ Error string }{Error: fmt.Errorf("contact the system administrator to activate account").Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
		sess, err := members.globalSessions.SessionStart(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer sess.SessionRelease(w)
		sess.Set("useremail", user.Email)
		http.SetCookie(w, &http.Cookie{Name: os.Getenv("Session_Cookie"), Value: sess.SessionID(), Path: "/", HttpOnly: false, Secure: true})
		json.NewEncoder(w).Encode(user)
		return
	} else if strings.EqualFold(r.URL.Path, "/logout") {
		members.globalSessions.SessionDestroy(w, r)
	} else if strings.EqualFold(r.URL.Path, "/member") {
		switch r.Method {
		case http.MethodPost:
			{
				var newmember Member
				err := json.NewDecoder(r.Body).Decode(&newmember)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				newmember.Id = uuid.NewString()
				u, err := members.Add(&newmember)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodGet:
			{
				result := make([]Member, 0)
				for _, m := range members.members {
					if m.Role != 1 {
						result = append(result, *m)
					}
				}
				json.NewEncoder(w).Encode(result)
				return
			}
		case http.MethodDelete:
			{
				var newmember Member
				err := json.NewDecoder(r.Body).Decode(&newmember)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

				}
				u, err := members.delete(&newmember)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodPut:
			{
				updatemember := make(map[string]interface{}, 0)
				err := json.NewDecoder(r.Body).Decode(&updatemember)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := members.update(updatemember)
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
