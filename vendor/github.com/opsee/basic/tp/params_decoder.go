package tp

import (
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
)

func ParamsDecoder(key interface{}) DecodeFunc {
	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
		newContext := context.WithValue(ctx, key, p)
		return newContext, 0, nil
	}
}

func QueryDecoder(key interface{}) DecodeFunc {
	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, _ httprouter.Params) (context.Context, int, error) {
		newContext := context.WithValue(ctx, key, r.URL.Query())
		return newContext, 0, nil
	}
}
