package groups

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

type Group struct {
	Id          string `bson:"Id"`
	Name        string `bson:"Name"`
	Email       string `bson:"Email"`
	Description string `bson:"Description"`
	Passport    string `bson:"Passport"`
}

type Groups struct {
	groups []*Group
	db     *mongo.Database
}

const groupCollection = "group"

func NewGroups(db *mongo.Database) *Groups {
	groups := make([]*Group, 0)
	col := db.Collection(groupCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err.Error())
	} else {
		if err = result.All(context.TODO(), &groups); err != nil {
			log.Fatal("error loading groups data " + err.Error())
		}
	}
	return &Groups{groups: groups, db: db}
}

func (groups *Groups) add(newgroup *Group) (*Group, error) {
	bsonData, err := bson.Marshal(newgroup)
	if err != nil {
		return nil, fmt.Errorf("error processing group details")
	}
	col := groups.db.Collection(groupCollection)
	_, err = col.InsertOne(context.TODO(), bsonData)
	if err != nil {
		return nil, fmt.Errorf("error registering user")
	}
	go func() {
		groups.groups = append(groups.groups, newgroup)
	}()
	return newgroup, nil
}

func (groups *Groups) delete(oldgroup *Group) (*Group, error) {
	for x, group := range groups.groups {
		if strings.EqualFold(group.Id, oldgroup.Id) {
			bsonData, err := bson.Marshal(oldgroup)
			if err != nil {
				return nil, fmt.Errorf("error processing group details")
			}
			col := groups.db.Collection(groupCollection)
			_, err = col.DeleteOne(context.TODO(), bsonData)
			if err != nil {
				return nil, fmt.Errorf("error deleting user")
			}
			go func() {
				groups.groups = append(groups.groups[:x], groups.groups[x+1:]...)
			}()
			return oldgroup, nil
		}
	}
	return nil, fmt.Errorf("group account does not exists")
}

func (groups *Groups) update(update map[string]interface{}) (*Group, error) {
	usr := Group{}
	for key, value := range update {
		field := reflect.ValueOf(&usr).Elem().FieldByName(key)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(value)
			if field.Type() == val.Type() {
				field.Set(val)
			} else {
				return nil, fmt.Errorf("Type mismatch for field %s", key)
			}
		}
	}
	for _, group := range groups.groups {
		if strings.EqualFold(group.Id, usr.Id) {
			update := bson.M{}
			bsonData, err := bson.Marshal(usr)
			if err != nil {
				return nil, fmt.Errorf("error updating group %s", err)
			}
			bson.Unmarshal(bsonData, &update)
			col := groups.db.Collection(groupCollection)
			_, err = col.UpdateOne(context.TODO(), bson.M{"Id": usr.Id}, bson.M{"$set": update})
			if err != nil {
				return nil, fmt.Errorf("error updating group %s", err)
			}
			*group = usr
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("group account does not exists")
}

func (groups *Groups) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/group") {
		switch r.Method {
		case http.MethodPost:
			{
				var newgroup Group
				err := json.NewDecoder(r.Body).Decode(&newgroup)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				newgroup.Id = uuid.NewString()
				u, err := groups.add(&newgroup)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodGet:
			{

				result := make([]Group, 0)
				for _, m := range groups.groups {
					result = append(result, *m)
				}
				json.NewEncoder(w).Encode(result)
				return
			}
		case http.MethodDelete:
			{
				var newgroup Group
				err := json.NewDecoder(r.Body).Decode(&newgroup)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

				}
				u, err := groups.delete(&newgroup)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodPut:
			{
				updategroup := make(map[string]interface{}, 0)
				err := json.NewDecoder(r.Body).Decode(&updategroup)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := groups.update(updategroup)
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
