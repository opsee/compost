package composter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/opsee/basic/schema"
	"github.com/opsee/basic/tp"
	log "github.com/opsee/logrus"
	"github.com/opsee/vaper"
	"golang.org/x/net/context"
)

var (
	errUnknown = errors.New("unknown error.")
)

func (s *Composter) StartHTTP(addr string) {
	http.ListenAndServe(addr, s.router)
}

func (s *Composter) initHTTP() {
	router := tp.NewHTTPRouter(context.Background())

	router.CORS(
		[]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		[]string{`https?://localhost:8080`, `https?://localhost:8008`, `https://(.+)?(opsy\.co|opsee\.co|opsee\.com)`, `https?://coreys-mbp-8:\d+`},
	)

	// graph q l
	router.Handle("POST", "/graphql", []tp.DecodeFunc{
		tp.AuthorizationDecodeFunc(userKey, schema.User{}),
		tp.RequestDecodeFunc(requestKey, GraphQLRequest{}),
	}, s.graphQL())
	router.Handle("POST", "/admin/graphql", []tp.DecodeFunc{
		s.authorizationDecodeFunc(),
		tp.RequestDecodeFunc(requestKey, GraphQLRequest{}),
	}, s.adminGraphQL())

	// fileserver for static things
	router.Handler("GET", "/static/*stuff", http.StripPrefix("/static/", http.FileServer(http.Dir("/static"))))

	// set a big timeout bc aws be slow
	router.Timeout(5 * time.Minute)

	s.router = router
}

func (s *Composter) authorizationDecodeFunc() tp.DecodeFunc {
	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
		header := r.Header.Get("authorization")
		if header == "" {
			return ctx, http.StatusUnauthorized, nil
		}

		var (
			authType string
			token    string
		)

		_, err := fmt.Sscanf(header, "%s %s", &authType, &token)
		if err != nil || token == "" {
			return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization header is malformed.")
		}

		if strings.ToLower(authType) != "bearer" {
			return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization type not supported.")
		}

		decoded, err := vaper.Unmarshal(token)
		if err != nil {
			log.WithError(err).Error("error decoding bearer token")
			return ctx, http.StatusUnauthorized, fmt.Errorf("Authorization token decode error.")
		}

		user := &schema.User{}
		err = decoded.Reify(user)
		if err != nil {
			return ctx, http.StatusUnauthorized, fmt.Errorf("authorization token unmarshal error.")
		}

		err = user.Validate()
		if err != nil {
			return ctx, http.StatusUnauthorized, err
		}

		return context.WithValue(ctx, userKey, user), 0, nil
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
