package districts

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

type District struct {
	Id          string `bson:"Id"`
	Name        string `bson:"Name"`
	Email       string `bson:"Email"`
	Description string `bson:"Description"`
	Passport    string `bson:"Passport"`
}

type Districts struct {
	districts []*District
	db        *mongo.Database
}

const districtCollection = "district"

func NewDistricts(db *mongo.Database) *Districts {
	districts := make([]*District, 0)
	col := db.Collection(districtCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err.Error())
	} else {
		if err = result.All(context.TODO(), &districts); err != nil {
			log.Fatal("error loading districts data " + err.Error())
		}
	}
	return &Districts{districts: districts, db: db}
}

func (districts *Districts) add(newdistrict *District) (*District, error) {
	bsonData, err := bson.Marshal(newdistrict)
	if err != nil {
		return nil, fmt.Errorf("error processing district details")
	}
	col := districts.db.Collection(districtCollection)
	_, err = col.InsertOne(context.TODO(), bsonData)
	if err != nil {
		return nil, fmt.Errorf("error registering user")
	}
	go func() {
		districts.districts = append(districts.districts, newdistrict)
	}()
	return newdistrict, nil
}

func (districts *Districts) delete(olddistrict *District) (*District, error) {
	for x, district := range districts.districts {
		if strings.EqualFold(district.Id, olddistrict.Id) {
			bsonData, err := bson.Marshal(olddistrict)
			if err != nil {
				return nil, fmt.Errorf("error processing district details")
			}
			col := districts.db.Collection(districtCollection)
			_, err = col.DeleteOne(context.TODO(), bsonData)
			if err != nil {
				return nil, fmt.Errorf("error deleting user")
			}
			go func() {
				districts.districts = append(districts.districts[:x], districts.districts[x+1:]...)
			}()
			return olddistrict, nil
		}
	}
	return nil, fmt.Errorf("district account does not exists")
}

func (districts *Districts) update(update map[string]interface{}) (*District, error) {
	var usr District
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
		if strings.EqualFold(v, "Id") {
			req += 1
		}
		if i == len(fields)-1 && req != 1 {
			return nil, fmt.Errorf("update missing important fields")
		}
	}
	for _, district := range districts.districts {
		if strings.EqualFold(district.Id, usr.Id) {

			col := districts.db.Collection(districtCollection)
			_, err := col.UpdateByID(context.TODO(), usr.Id, usr)
			if err != nil {
				return nil, fmt.Errorf("error updating district")
			}
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("district account does not exists")
}

func (districts *Districts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/district") {
		switch r.Method {
		case http.MethodPost:
			{
				var newdistrict District
				err := json.NewDecoder(r.Body).Decode(&newdistrict)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				newdistrict.Id = uuid.NewString()
				u, err := districts.add(&newdistrict)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodGet:
			{
				result := make([]District, 0)
				for _, m := range districts.districts {
					result = append(result, *m)
				}
				json.NewEncoder(w).Encode(result)
				return
			}
		case http.MethodDelete:
			{
				var newdistrict District
				err := json.NewDecoder(r.Body).Decode(&newdistrict)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

				}
				u, err := districts.delete(&newdistrict)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodPut:
			{
				updatedistrict := make(map[string]interface{}, 0)
				err := json.NewDecoder(r.Body).Decode(&updatedistrict)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := districts.update(updatedistrict)
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
