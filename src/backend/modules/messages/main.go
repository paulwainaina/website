package messages

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

type Message struct {
	Id          string `bson:"Id"`
	Email       string `bson:"Email"`
	Description string `bson:"Description"`
}

type Messages struct {
	messages []*Message
	db       *mongo.Database
}

const messageCollection = "message"

func NewMessages(db *mongo.Database) *Messages {
	messages := make([]*Message, 0)
	col := db.Collection(messageCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err.Error())
	} else {
		if err = result.All(context.TODO(), &messages); err != nil {
			log.Fatal("error loading messages data " + err.Error())
		}
	}
	return &Messages{messages: messages, db: db}
}

func (messages *Messages) add(newmessage *Message) (*Message, error) {
	bsonData, err := bson.Marshal(newmessage)
	if err != nil {
		return nil, fmt.Errorf("error processing message details")
	}
	col := messages.db.Collection(messageCollection)
	_, err = col.InsertOne(context.TODO(), bsonData)
	if err != nil {
		return nil, fmt.Errorf("error registering user")
	}
	go func() {
		messages.messages = append(messages.messages, newmessage)
	}()
	return newmessage, nil
}

func (messages *Messages) delete(oldmessage *Message) (*Message, error) {
	for x, message := range messages.messages {
		if strings.EqualFold(message.Id, oldmessage.Id) {
			bsonData, err := bson.Marshal(oldmessage)
			if err != nil {
				return nil, fmt.Errorf("error processing message details")
			}
			col := messages.db.Collection(messageCollection)
			_, err = col.DeleteOne(context.TODO(), bsonData)
			if err != nil {
				return nil, fmt.Errorf("error deleting user")
			}
			go func() {
				messages.messages = append(messages.messages[:x], messages.messages[x+1:]...)
			}()
			return oldmessage, nil
		}
	}
	return nil, fmt.Errorf("message account does not exists")
}

func (messages *Messages) update(update map[string]interface{}) (*Message, error) {
	var usr Message
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
	for _, message := range messages.messages {
		if strings.EqualFold(message.Id, usr.Id) {

			col := messages.db.Collection(messageCollection)
			_, err := col.UpdateByID(context.TODO(), usr.Id, usr)
			if err != nil {
				return nil, fmt.Errorf("error updating message")
			}
			return &usr, nil
		}
	}
	return nil, fmt.Errorf("message account does not exists")
}

func (messages *Messages) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.URL.Path, "/message") {
		switch r.Method {
		case http.MethodPost:
			{
				var newmessage Message
				err := json.NewDecoder(r.Body).Decode(&newmessage)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				newmessage.Id=uuid.NewString()
				u, err := messages.add(&newmessage)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodGet:
			{
				if len(messages.messages) == 0 {
					json.NewEncoder(w).Encode("{error: no users found}")
					return
				}
				json.NewEncoder(w).Encode(fmt.Sprintf("{messages: %v}", messages.messages))
				return
			}
		case http.MethodDelete:
			{
				var newmessage Message
				err := json.NewDecoder(r.Body).Decode(&newmessage)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

				}
				u, err := messages.delete(&newmessage)
				if err != nil {
					json.NewEncoder(w).Encode(fmt.Sprintf("{error: %s}", err.Error()))
					return
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		case http.MethodPut:
			{
				updatemessage := make(map[string]interface{}, 0)
				err := json.NewDecoder(r.Body).Decode(&updatemessage)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := messages.update(updatemessage)
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
