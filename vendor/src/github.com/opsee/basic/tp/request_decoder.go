package tp

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type RequestValidator interface {
	Validate() error
}

func RequestDecodeFunc(requestKey int, requestType interface{}) DecodeFunc {
	return func(ctx context.Context, rw http.ResponseWriter, r *http.Request, _ httprouter.Params) (context.Context, int, error) {
		var (
			reflectType      = reflect.TypeOf(requestType)
			requestValue     = reflect.New(reflectType)
			requestInterface interface{}
			err              error
		)

		if r.Method == "GET" {
			requestInterface, err = requestDecodeGET(r, reflectType, requestValue)
		} else {
			requestInterface, err = requestDecodeBody(r, reflectType, requestValue)
		}

		if err != nil {
			return ctx, http.StatusInternalServerError, err
		}

		request, ok := requestInterface.(RequestValidator)
		if !ok {
			return ctx, http.StatusInternalServerError, fmt.Errorf("Failed type assertion for type: %#v", requestType)
		}

		err = request.Validate()
		if err != nil {
			return ctx, http.StatusBadRequest, err
		}

		return context.WithValue(ctx, requestKey, request), 0, nil
	}
}

func requestDecodeBody(r *http.Request, reflectType reflect.Type, requestValue reflect.Value) (interface{}, error) {
	request := requestValue.Interface()
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(request)
	if err != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("Malformed request body: %s", string(body))
	}

	return request, nil
}

func requestDecodeGET(r *http.Request, reflectType reflect.Type, requestValue reflect.Value) (interface{}, error) {
	numField := reflectType.NumField()
	for i := 0; i < numField; i += 1 {
		var nameOrTag string

		f := reflectType.Field(i)
		jtag := f.Tag.Get("json")

		jtagparts := strings.SplitN(jtag, ",", 2)
		if len(jtagparts) > 0 && jtagparts[0] != "" {
			nameOrTag = jtagparts[0]
		} else {
			nameOrTag = f.Name
		}

		err := reflectSet(f.Type, reflect.Indirect(requestValue).FieldByName(f.Name), r.FormValue(nameOrTag))
		if err != nil {
			return nil, fmt.Errorf("Error parsing field %s - %s", nameOrTag, err.Error())
		}
	}

	return requestValue.Interface(), nil
}

func reflectSet(ft reflect.Type, val reflect.Value, formVal string) error {
	if val.CanSet() && formVal != "" {
		switch ft.Kind() {
		case reflect.Bool:
			boolVal := false

			if strings.ToLower(formVal) != "false" {
				boolVal = true
			}

			val.SetBool(boolVal)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(formVal, 10, 64)
			if err != nil {
				return fmt.Errorf("bad integer value: %s", formVal)
			}

			val.SetInt(intVal)

		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(formVal, 64)
			if err != nil {
				return fmt.Errorf("bad float value: %s", formVal)
			}

			val.SetFloat(floatVal)

		case reflect.Slice:
			stringSlice := strings.Split(formVal, ",")
			sliceVal := reflect.MakeSlice(ft, len(stringSlice), len(stringSlice))

			for si, sv := range stringSlice {
				reflectSet(ft.Elem(), sliceVal.Index(si), sv)
			}

			val.Set(sliceVal)

		case reflect.String:
			val.SetString(formVal)
		}
	}

	return nil
}
