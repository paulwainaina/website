package members

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Member struct {
	Id              string `bson:"Id"`
	Name            string `bson:"Name"`
	Email           string `bson:"Email"`
	Contacts        string `bson:"Contacts`
	DateofBirth     string `bson:"DateofBirth"`
	DateofBaptism   string `bson:"DateofBaptism"`
	DateofCatechism string `bson:"DateofCatechism"`
	District        string `bson:"District"`
	Groups          string `bson:"Groups"`
	Passport        string `bson:"Passport"`
}

type Members struct {
	members []*Member
	db      *mongo.Database
}

const memberCollection = "member"

func NewMembers(db *mongo.Database) *Members {
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
	return &Members{members: members, db: db}
}

func (members *Members) add(newmember *Member) (*Member, error) {
	for _, member := range members.members {
		if strings.EqualFold(member.Email, newmember.Email) {
			return nil, fmt.Errorf("member already exists")
		}
	}
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
	var usr Member
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
	for _, member := range members.members {
		if strings.EqualFold(member.Email, usr.Email) && strings.EqualFold(member.Id, usr.Id) {

			col := members.db.Collection(memberCollection)
			_, err := col.UpdateByID(context.TODO(), usr.Id, usr)
			if err != nil {
				return nil, fmt.Errorf("error updating member")
			}
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("member account does not exists")
}

func (members *Members) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/member") {
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
				u, err := members.add(&newmember)
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
					result = append(result, *m)
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
