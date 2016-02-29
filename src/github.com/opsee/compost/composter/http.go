package composter

import (
	"errors"
	"github.com/opsee/basic/schema"
	"github.com/opsee/basic/tp"
	"golang.org/x/net/context"
	"net/http"
	"time"
)

var (
	errUnknown = errors.New("unknown error.")
)

func (s *Composter) StartHTTP(addr string) {
	router := tp.NewHTTPRouter(context.Background())

	// graph q l
	router.Handle("POST", "/graphql", decoders(schema.User{}, GraphQLRequest{}), s.graphQL())
	router.Handle("POST", "/admin/graphql", decoders(schema.User{}, GraphQLRequest{}), s.adminGraphQL())

	// fileserver for static things
	router.Handler("GET", "/static/*stuff", http.StripPrefix("/static/", http.FileServer(http.Dir("/static"))))

	// set a big timeout bc aws be slow
	router.Timeout(5 * time.Minute)

	http.ListenAndServe(addr, router)
}

func decoders(userType interface{}, requestType interface{}) []tp.DecodeFunc {
	return []tp.DecodeFunc{
		tp.AuthorizationDecodeFunc(userKey, userType),
		tp.RequestDecodeFunc(requestKey, requestType),
	}
}

func (s *Composter) graphQL() tp.HandleFunc {
	return func(ctx context.Context) (interface{}, int, error) {
		_, ok := ctx.Value(userKey).(*schema.User)
		if !ok {
			return nil, http.StatusUnauthorized, errDecodeUser
		}

		response, err := s.Compost(ctx, s.Schema)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		return response, http.StatusOK, nil
	}
}

func (s *Composter) adminGraphQL() tp.HandleFunc {
	return func(ctx context.Context) (interface{}, int, error) {
		_, ok := ctx.Value(userKey).(*schema.User)
		if !ok {
			return nil, http.StatusUnauthorized, errDecodeUser
		}

		response, err := s.Compost(ctx, s.AdminSchema)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		return response, http.StatusOK, nil
	}
}
