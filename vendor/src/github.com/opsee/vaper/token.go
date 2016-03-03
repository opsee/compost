package vaper

import (
	"encoding/json"
	"errors"
	"github.com/dvsekhvalnov/jose2go"
	"reflect"
	"time"
)

type Token map[string]interface{}

const (
	Algorithm  = jose.A128GCMKW
	Encryption = jose.A128GCM
)

var vapeKey []byte

func Init(sharedKey []byte) {
	vapeKey = sharedKey
}

func New(thing interface{}, sub string, iat, exp time.Time) *Token {
	token := make(Token)

	t := reflectValue(thing)
	for i := 0; i < t.NumField(); i++ {
		tag := t.Type().Field(i).Tag.Get("token")
		if tag != "" {
			token[tag] = t.Field(i).Interface()
		}
	}

	token["sub"] = sub
	token["iat"] = iat.Unix()
	token["exp"] = exp.Unix()

	return &token
}

func (token *Token) Marshal() (string, error) {
	json, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	return jose.Encrypt(string(json), Algorithm, Encryption, vapeKey)
}

func (token *Token) Reify(thing interface{}) error {
	t := reflectValue(thing)

	for i := 0; i < t.NumField(); i++ {
		tag := t.Type().Field(i).Tag.Get("token")
		kind := t.Field(i).Kind()

		val, ok := (*token)[tag]
		if !ok {
			continue
		}

		switch val.(type) {
		case float64: // a special case for json turning things into floats
			switch kind {
			case reflect.Int, reflect.Int32, reflect.Int64:
				t.Field(i).SetInt(int64(val.(float64)))
			default:
				t.Field(i).Set(reflect.ValueOf(val))
			}
		case string: // a special case for timestamps
			if kind == reflect.Struct {
				date, err := time.Parse(time.RFC3339, val.(string))
				if err != nil {
					return err
				}
				t.Field(i).Set(reflect.ValueOf(date))
			} else {
				t.Field(i).Set(reflect.ValueOf(val))
			}
		default:
			t.Field(i).Set(reflect.ValueOf(val))
		}
	}

	return nil
}

func Unmarshal(tokenString string) (*Token, error) {
	payload, headers, err := jose.Decode(tokenString, vapeKey)
	if err != nil {
		return nil, err
	}

	if headers["alg"] != Algorithm {
		return nil, errors.New("token alg does not match")
	}

	if headers["enc"] != Encryption {
		return nil, errors.New("token enc does not match")
	}

	token := make(Token)
	err = json.Unmarshal([]byte(payload), &token)
	if err != nil {
		return nil, err
	}

	token["exp"] = int64(token["exp"].(float64))
	token["iat"] = int64(token["iat"].(float64))

	now := time.Now().UTC()
	exp := time.Unix(token["exp"].(int64), 0)
	iat := time.Unix(token["iat"].(int64), 0)

	if now.Before(exp) != true {
		return nil, errors.New("token expired")
	}

	if iat.Before(now) != true {
		return nil, errors.New("token issued after now")
	}

	return &token, nil
}

func Verify(tokenString string) error {
	_, err := Unmarshal(tokenString)
	return err
}

func reflectValue(obj interface{}) reflect.Value {
	var val reflect.Value

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		val = reflect.ValueOf(obj).Elem()
	} else {
		val = reflect.ValueOf(obj)
	}

	return val
}
