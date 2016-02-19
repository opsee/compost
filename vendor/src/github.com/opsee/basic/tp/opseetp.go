package tp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type DecodeFunc func(context.Context, http.ResponseWriter, *http.Request, httprouter.Params) (context.Context, int, error)
type HandleFunc func(context.Context) (interface{}, int, error)
type EncodeFunc func(interface{}) ([]byte, error)

type Router struct {
	router     *httprouter.Router
	rootCtx    context.Context
	encoders   map[string]EncodeFunc
	encoderMut *sync.Mutex
	timeout    time.Duration
	corsFunc   DecodeFunc
}

type MessageResponse struct {
	Message string `json:"message"`
}

type requestForwarder struct {
	response interface{}
	status   int
	err      error
}

const (
	defaultTimeout     = 5 * time.Second
	defaultContentType = "application/json"
	defaultAccept      = "application/json"
)

func NewHTTPRouter(rootCtx context.Context) *Router {
	r := &Router{
		router:     httprouter.New(),
		rootCtx:    rootCtx,
		encoders:   make(map[string]EncodeFunc),
		encoderMut: &sync.Mutex{},
		timeout:    defaultTimeout,
	}

	r.router.HandleMethodNotAllowed = true
	r.router.PanicHandler = panicHandler
	r.router.RedirectTrailingSlash = false

	r.Handle("OPTIONS", "/*any", []DecodeFunc{decodeIdentity}, okHandler)
	r.Handle("GET", "/health", []DecodeFunc{decodeIdentity}, okHandler)

	r.Encoder(defaultAccept, func(response interface{}) ([]byte, error) {
		return json.Marshal(response)
	})

	return r
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(rw, req)
}

func (r *Router) Handle(method, path string, decoders []DecodeFunc, handler HandleFunc) {
	r.router.Handle(method, path, r.wrapHandler(decoders, handler))
}

func (r *Router) Handler(method, path string, handler http.Handler) {
	r.router.Handler(method, path, handler)
}

func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.router.HandlerFunc(method, path, handler)
}

func (r *Router) Encoder(contentType string, encodeFunc EncodeFunc) {
	r.encoderMut.Lock()
	defer r.encoderMut.Unlock()
	r.encoders[contentType] = encodeFunc
}

func (r *Router) CORS(methods, origins []string) {
	r.corsFunc = CORSRegexpDecodeFunc(methods, origins)
}

func (r *Router) Timeout(timeout time.Duration) {
	r.timeout = timeout
}

func (r *Router) wrapHandler(decoders []DecodeFunc, handler HandleFunc) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		var (
			ctx    context.Context
			cancel context.CancelFunc
			status int
			err    error
		)

		ctx, cancel = context.WithTimeout(r.rootCtx, r.timeout)
		defer cancel()

		if r.corsFunc != nil {
			ctx, status, err = r.corsFunc(ctx, rw, req, params)
			if err != nil {
				serverError(rw, req, status, err)
				return
			}
		}

		for _, decoder := range decoders {
			ctx, status, err = decoder(ctx, rw, req, params)
			if err != nil {
				serverError(rw, req, status, err)
				return
			}
		}

		forwardChan := make(chan requestForwarder)
		go func() {
			defer close(forwardChan)
			response, status, err := handler(ctx)
			forwardChan <- requestForwarder{response, status, err}
		}()

		select {
		case rf := <-forwardChan:
			if rf.err != nil {
				serverError(rw, req, rf.status, rf.err)
				return
			}

			contentType, encoder := r.getEncoder(req.Header.Get("accept"))
			encodedResponse, err := encoder(rf.response)
			if err != nil {
				serverError(rw, req, http.StatusInternalServerError, err)
				return
			}

			rw.Header().Set("Content-Type", contentType)
			rw.WriteHeader(rf.status)
			rw.Write(encodedResponse)
			log.WithFields(log.Fields{
				"status": rf.status,
				"path":   req.URL.RequestURI(),
				"method": req.Method,
			}).Info("tp request")

		case <-ctx.Done():
			msg, _ := json.Marshal(MessageResponse{"Backend service unavailable."})
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Write(msg)
		}
	}
}

func (r *Router) getEncoder(accept string) (string, EncodeFunc) {
	contentTypes := strings.Split(accept, ",")
	for _, v := range contentTypes {
		ct := strings.Split(v, ";")
		if ct[0] != "" {
			contentType := strings.TrimSpace(ct[0])
			enc, ok := r.encoders[contentType]
			if ok {
				return contentType, enc
			}
		}
	}
	return defaultContentType, r.encoders[defaultAccept]
}

func serverError(rw http.ResponseWriter, req *http.Request, status int, err error) {
	log.WithError(err).WithField("request", *req).WithError(err).Error("error")

	var message string
	if status == 0 || status == http.StatusInternalServerError {
		message = "An unexpeced error happened."
	} else {
		message = err.Error()
	}

	msg, _ := json.Marshal(MessageResponse{message})
	rw.Header().Set("Content-Type", defaultContentType)

	rw.WriteHeader(status)
	rw.Write(msg)
}

func panicHandler(rw http.ResponseWriter, req *http.Request, data interface{}) {
	serverError(rw, req, http.StatusInternalServerError, fmt.Errorf("panic data: %#v", data))
}

func decodeIdentity(ctx context.Context, rw http.ResponseWriter, req *http.Request, params httprouter.Params) (context.Context, int, error) {
	return ctx, 0, nil
}

func okHandler(ctx context.Context) (interface{}, int, error) {
	return map[string]bool{"ok": true}, http.StatusOK, nil
}
