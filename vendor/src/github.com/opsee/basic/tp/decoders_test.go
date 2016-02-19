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

type testRequestValidator struct {
	Count      int `json:"count"`
	Int32Slice []int32
	StrSlice   []string `json:"cool_string_slice"`
	Name       string   `json:",omitempty"`
}

func (rv *testRequestValidator) Validate() error {
	if rv.Count == 777 {
		return fmt.Errorf("count is 777")
	}

	return nil
}

func (u *user) Validate() error {
	if u.Id == 1 {
		return nil
	}

	return fmt.Errorf("user not authorized")
}

func TestCORS(t *testing.T) {
	router := NewHTTPRouter(context.Background())
	router.CORS(
		[]string{"GET"},
		[]string{`http://(\w+\.)?(opsy\.co|opsee\.co|opsee)`},
	)

	router.Handle("GET", "/", []DecodeFunc{}, func(ctx context.Context) (interface{}, int, error) {
		return map[string]interface{}{"ok": true}, http.StatusOK, nil
	})

	req, err := http.NewRequest("GET", "http://potata.opsee/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "GET", rw.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "", rw.Header().Get("Access-Control-Allow-Origin"))

	req, err = http.NewRequest("GET", "http://potata.opsee/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://potata.opsee")

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "http://potata.opsee", rw.Header().Get("Access-Control-Allow-Origin"))
}

func TestParams(t *testing.T) {
	paramsKey := 0

	router := NewHTTPRouter(context.Background())
	paramDecoder := ParamsDecoder(paramsKey)

	type response struct {
		Id string `json:"id"`
	}

	router.Handle("GET", "/foo/:id", []DecodeFunc{paramDecoder}, func(ctx context.Context) (interface{}, int, error) {
		paramsValue := ctx.Value(paramsKey)
		if paramsValue == nil {
			return nil, http.StatusBadRequest, fmt.Errorf("No parameters found in context")
		}

		params, ok := paramsValue.(httprouter.Params)
		if !ok {
			return nil, http.StatusBadRequest, fmt.Errorf("Error reading params in context")
		}

		response := response{params.ByName("id")}

		return response, http.StatusOK, nil
	})

	req, err := http.NewRequest("GET", "http://localhost/foo/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	r := &response{}
	err = json.NewDecoder(rw.Body).Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "1", r.Id)

	req, err = http.NewRequest("GET", "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAuthorization(t *testing.T) {
	userKey := 0

	router := NewHTTPRouter(context.Background())
	authDecoder := AuthorizationDecodeFunc(userKey, user{})

	router.Handle("GET", "/", []DecodeFunc{authDecoder}, func(ctx context.Context) (interface{}, int, error) {
		return ctx.Value(userKey), http.StatusOK, nil
	})

	req, err := http.NewRequest("GET", "http://potata.opsee/", nil)
	if err != nil {
		t.Fatal(err)
	}

	token := base64.StdEncoding.EncodeToString([]byte(`{"id": 1, "email": "cliff@leaninto.it"}`))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", token))

	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)

	u := &user{}
	err = json.NewDecoder(rw.Body).Decode(u)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, u.Id, 1)
	assert.Equal(t, u.Email, "cliff@leaninto.it")

	req, err = http.NewRequest("GET", "http://potata.opsee/", nil)
	if err != nil {
		t.Fatal(err)
	}

	token = base64.StdEncoding.EncodeToString([]byte(`{"id": 2, "email": "cliff@leaninto.it"}`))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", token))

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestRequest(t *testing.T) {
	requestKey := 0

	router := NewHTTPRouter(context.Background())
	requestDecoder := RequestDecodeFunc(requestKey, testRequestValidator{})

	router.Handle("GET", "/", []DecodeFunc{requestDecoder}, func(ctx context.Context) (interface{}, int, error) {
		return ctx.Value(requestKey), http.StatusOK, nil
	})
	router.Handle("PUT", "/", []DecodeFunc{requestDecoder}, func(ctx context.Context) (interface{}, int, error) {
		return ctx.Value(requestKey), http.StatusOK, nil
	})

	req, err := http.NewRequest("PUT", "http://potata.opsee/", bytes.NewBufferString(`{"count": 1}`))
	if err != nil {
		t.Fatal(err)
	}

	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)

	rv := &testRequestValidator{}
	err = json.NewDecoder(rw.Body).Decode(rv)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, rv.Count)

	req, err = http.NewRequest("PUT", "http://potata.opsee/", bytes.NewBufferString(`{"count": 777}`))
	if err != nil {
		t.Fatal(err)
	}

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusBadRequest, rw.Code)

	req, err = http.NewRequest("GET", "http://potata.opsee/?count=777", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusBadRequest, rw.Code)

	req, err = http.NewRequest("GET", "http://potata.opsee/?count=222&Name=rip&cool_string_slice=1,2,egg&Int32Slice=9,8,7", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)

	rv = &testRequestValidator{}
	err = json.NewDecoder(rw.Body).Decode(rv)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 222, rv.Count)
	assert.Equal(t, "rip", rv.Name)
	assert.Equal(t, []int32{int32(9), int32(8), int32(7)}, rv.Int32Slice)
	assert.Equal(t, []string{"1", "2", "egg"}, rv.StrSlice)
}
