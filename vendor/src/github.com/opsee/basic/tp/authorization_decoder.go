package tp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
	"reflect"
	"strings"
)

type AuthorizationValidator interface {
	Validate() error
}

func AuthorizationDecodeFunc(tokenKey int, tokenUnmarshaler interface{}) DecodeFunc {
	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
		header := r.Header.Get("authorization")
		if header == "" {
			return ctx, http.StatusUnauthorized, nil
		}

		var (
			authType = new(string)
			token    = new(string)
		)

		_, err := fmt.Sscanf(header, "%s %s", authType, token)
		if err != nil || *token == "" {
			return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization header is malformed.")
		}

		switch strings.ToLower(*authType) {
		case "basic":
			return decodeBasic(ctx, token, tokenKey, tokenUnmarshaler)
		default:
			return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization type not supported.")
		}
	}
}

func decodeBasic(ctx context.Context, token *string, tokenKey int, tokenType interface{}) (context.Context, int, error) {
	jsonblob, err := base64.StdEncoding.DecodeString(*token)
	if err != nil {
		return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization token decode error.")
	}

	unmarshaler := reflect.New(reflect.TypeOf(tokenType)).Interface().(AuthorizationValidator)
	err = json.Unmarshal(jsonblob, unmarshaler)
	if err != nil {
		return ctx, http.StatusUnauthorized, fmt.Errorf("authorization token unmarshal error.")
	}

	err = unmarshaler.Validate()
	if err != nil {
		return ctx, http.StatusUnauthorized, err
	}

	return context.WithValue(ctx, tokenKey, unmarshaler), 0, nil
}
