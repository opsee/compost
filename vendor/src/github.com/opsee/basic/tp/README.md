tp
=======

HTTP router for your go services with middleware.

Usage
-----

Create a router instance with your root context, and register handlers. Handlers can be registered with a method, path, and slice of decoders (middleware) that run before the handler:

```go

struct Service {
        db *db
        queue *queue
}

const (
        serviceKey = iota
        reqKey
)

func (s *Service) ListInstances(request *store.InstancesRequest) (*store.InstancesResponse, err) {
        return s.db.ListInstances(request.CustomerId)
}

func instancesDecoder(ctx context.Context, rw http.ResponseWriter, r *http.Request, params httprouter.Params) (context.Contex, int, error) {
	customerId := r.Header.Get("Customer-Id")
	if customerId == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("customer id must be provided.")
	}

	req := &store.InstancesRequest{
		CustomerId: customerId,
		Type:       params.ByName("type"),
	}

        ctx = context.WithValue(ctx, reqKey, req)
        return ctx, 0, nil
}

func instancesHandler(ctx context.Context) (interface{}, int, error) {
        service := ctx.Value(serviceKey).(*Service)
        request := ctx.Value(reqKey).(*store.InstancesRequest)

	response, err := service.ListInstances(request)
	if err != nil {
		return nil, 0, err
	}

	return response, http.StatusOK, nil
}

func main() {
        router := tp.NewHTTPRouter(
                context.WithValue(                // the root context for all requests
                        context.Background(),
                        serviceKey,
                        &Service{mydb, myqueue},
                ),
        )

        router.Handle("GET", "/instances", []tp.DecodeFunc{instancesDecoder}, instancesHandler) // add a route
        router.Timeout(5 * time.Second) // add a timeout for your backend
        http.ListenAndServe(addr, router)
}

```

Decoders (middleware)
---------------------

Included middleware so far are:

- CORS
- Authorization header

```go
// cors
router := tp.NewHTTPRouter(context.Background())
corsDecoder := CORSRegexpDecodeFunc([]string{"GET", "POST"}, []string{`https?://(\w\.)?opsee\.com`})
router.Handle("GET", "/", []DecodeFunc{corsDecoder}, myHandler)

// authorization decoder
router := tp.NewHTTPRouter(context.Background())
authDecoder := AuthorizationDecodeFunc(userKey, User{})
router.Handle("GET", "/", []DecodeFunc{authDecoder}, myHandler)

// then in myHandler...
func myHandler(ctx context.Context) (interface{}, int, error) {
        user := ctx.Value(userKey).(*User)
        ...
}
```
