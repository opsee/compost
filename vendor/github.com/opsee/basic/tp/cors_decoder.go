package tp

import (
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
	"regexp"
	"strings"
)

func CORSRegexpDecodeFunc(methods, origins []string) DecodeFunc {
	originsRegexp := make([]*regexp.Regexp, len(origins))

	for i, o := range origins {
		originsRegexp[i] = regexp.MustCompile(o)
	}

	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, p httprouter.Params) (context.Context, int, error) {
		origin := r.Header.Get("Origin")

		header := rw.Header()
		header.Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
		header.Set("Access-Control-Allow-Headers", "Accept-Encoding,Authorization,Content-Type")
		header.Set("Access-Control-Max-Age", "1728000")

		for _, o := range originsRegexp {
			if o.MatchString(origin) {
				header.Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		return ctx, 0, nil
	}
}
