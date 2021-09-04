/**
 * @Time : 19/11/2019 10:41 AM
 * @Author : solacowa@gmail.com
 * @File : handler
 * @Software: GoLand
 */

package http

import (
	"context"
	"encoding/json"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/icowan/shorter/pkg/endpoint"
	"github.com/icowan/shorter/pkg/service"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	ErrCodeNotFound = errors.New("code is nil")
)

func NewHTTPHandler(endpoints endpoint.Endpoints, options map[string][]kithttp.ServerOption) http.Handler {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("./dist"))).Methods(http.MethodGet)
	r.Handle("/umi.js", http.FileServer(http.Dir("./dist"))).Methods(http.MethodGet)
	r.Handle("/umi.css", http.FileServer(http.Dir("./dist"))).Methods(http.MethodGet)
	r.Handle("/favicon.ico", http.FileServer(http.Dir("./dist"))).Methods(http.MethodGet)

	r.Handle("/{code}", kithttp.NewServer(
		endpoints.GetEndpoint,
		decodeGetRequest,
		encodeGetResponse,
		options["Get"]...)).Methods(http.MethodGet)

	r.Handle("/", kithttp.NewServer(
		endpoints.PostEndpoint,
		decodePostRequest,
		decodePostResponse,
		options["Post"]...)).Methods(http.MethodPost)

	return r
}

func decodePostRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.PostRequest
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &req); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return nil, errors.Wrap(err, service.ErrRedirectInvalid.Error())
	}
	return req, nil
}

func decodePostResponse(ctx context.Context, w http.ResponseWriter, response interface{}) (err error) {
	if f, ok := response.(endpoint.Failure); ok && f.Failed() != nil {
		ErrorEncoder(ctx, f.Failed(), w)
		return nil
	}
	err = json.NewEncoder(w).Encode(response)
	return
}

func decodeGetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	code, ok := vars["code"]
	if !ok {
		return nil, ErrCodeNotFound
	}
	req := endpoint.GetRequest{
		Code: code,
	}
	return req, nil
}

func encodeGetResponse(ctx context.Context, w http.ResponseWriter, response interface{}) (err error) {
	if f, ok := response.(endpoint.Failure); ok && f.Failed() != nil {
		ErrorRedirect(ctx, f.Failed(), w)
		return nil
	}
	resp := response.(endpoint.GetResponse)
	redirect := resp.Data.(*service.Redirect)
	http.Redirect(w, &http.Request{}, redirect.URL, http.StatusFound)
	return
}

func ErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	_ = json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func ErrorRedirect(_ context.Context, err error, w http.ResponseWriter) {
	http.Redirect(w, &http.Request{}, os.Getenv("SHORT_URI"), http.StatusFound)
}

func ErrorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

func err2code(err error) int {
	return http.StatusOK
}

type errorWrapper struct {
	Error string `json:"error"`
}
