package tp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testService struct {
	counter int
}

type incrementRequest struct {
	Amount int `json:amount`
}

type user struct {
	Id    int    `json:id`
	Email string `json:email`
}

const (
	serviceKey = iota
	requestKey
	userKey
)

func decodeUser(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
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
		return ctx, http.StatusUnauthorized, fmt.Errorf("authorization header in bad format")
	}

	jsonblob, err := base64.StdEncoding.DecodeString(*token)
	if err != nil {
		return ctx, http.StatusUnauthorized, fmt.Errorf("authorization token decode error")
	}

	user := &user{}
	err = json.Unmarshal(jsonblob, user)
	if err != nil {
		return ctx, http.StatusUnauthorized, fmt.Errorf("authorization token unmarshal error")
	}

	return context.WithValue(ctx, userKey, user), 0, nil
}

func decodeIncrement(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
	request := &incrementRequest{}
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(request)
	if err != nil {
		return ctx, http.StatusBadRequest, err
	}

	return context.WithValue(ctx, requestKey, request), 0, nil
}

func handleIncrement(ctx context.Context) (interface{}, int, error) {
	service := ctx.Value(serviceKey).(*testService)
	user, ok := ctx.Value(userKey).(*user)

	if !ok || user == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("user required to perform this action")
	}

	request := ctx.Value(requestKey).(*incrementRequest)
	service.counter = service.counter + request.Amount

	return map[string]int{"counter": service.counter}, http.StatusOK, nil
}

func TestService(t *testing.T) {
	service := &testService{}

	router := NewHTTPRouter(context.WithValue(context.Background(), serviceKey, service))
	router.Handle("PUT", "/increment", []DecodeFunc{decodeUser, decodeIncrement}, handleIncrement)

	req, err := http.NewRequest("PUT", "http://testservice/increment", bytes.NewBuffer([]byte(`{"amount": 5}`)))
	if err != nil {
		t.Fatal(err)
	}

	token := base64.StdEncoding.EncodeToString([]byte(`{"id": 1, "email": "cliff@leaninto.it"}`))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", token))

	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, 5, service.counter)
}
