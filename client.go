package restclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/go-resty/resty"
)

// Auth .
type Auth interface {
	Handle(request *http.Request)
}

// Client .
type Client interface {
	POST(path string, request interface{}, options ...Option) Result
	GET(path string, request interface{}, options ...Option) Result
	DELETE(path string, request interface{}, options ...Option) Result
}

// Option .
type Option func(request *http.Request)

// WithAuth add auth option
func WithAuth(auth Auth) Option {
	return func(request *http.Request) {
		auth.Handle(request)
	}
}

// WithJWToken .
func WithJWToken(token string) Option {
	return func(request *http.Request) {
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
}

// Result .
type Result interface {
	OK() bool
	Fail() bool
	Error() error
	Response() *resty.Response
	Value(key string, result interface{}) error
	Values() map[string]interface{}
}

type clientImpl struct {
	sync.RWMutex
	url  string // url
	auth Auth
}

type resultImpl struct {
	err    error
	resp   *resty.Response
	values map[string]interface{}
}

func newResult(err error, resp *resty.Response) Result {
	return &resultImpl{
		err:  err,
		resp: resp,
	}
}

func (result *resultImpl) Response() *resty.Response {
	return result.resp
}

func (result *resultImpl) extractValues() {
	if result.values != nil {
		return
	}

	if result.resp == nil {
		return
	}

	values := make(map[string]interface{})

	json.Unmarshal(result.resp.Body(), &values)

	result.values = values

}

func (result *resultImpl) OK() bool {
	return result.err == nil && result.resp.StatusCode() == http.StatusOK
}
func (result *resultImpl) Fail() bool {
	return !result.OK()
}

func (result *resultImpl) Error() error {

	if result.OK() {
		return nil
	}

	if result.err != nil {
		return result.err
	}

	if result.resp != nil {
		return fmt.Errorf("status code(%s) %s", result.resp.Status(), string(result.resp.Body()))
	}

	return nil
}

func (result *resultImpl) Value(key string, v interface{}) error {

	result.extractValues()

	data, ok := result.values[key]

	if !ok {
		return fmt.Errorf("unknown return value %s\n%s", key, string(result.resp.Body()))
	}

	buff, err := json.Marshal(data)

	if err != nil {
		return fmt.Errorf("unmarshal result(%s) err %s\n%s", key, err, string(result.resp.Body()))
	}

	if err := json.Unmarshal(buff, v); err != nil {
		return fmt.Errorf("unmarshal result(%s) err %s\n%s", key, err, string(buff))
	}

	return nil
}

func (result *resultImpl) Values() map[string]interface{} {
	return result.values
}

// New .
func New(url string) Client {
	return &clientImpl{
		url: url,
	}
}

func (client *clientImpl) POST(path string, request interface{}, options ...Option) Result {

	r := resty.R().SetBody(request) //.Post(fmt.Sprintf("%s/%s", client.url, path))

	for _, option := range options {
		option(r.RawRequest)
	}

	url, err := client.checkURL(fmt.Sprintf("%s%s", client.url, path))

	if err != nil {
		return newResult(err, nil)
	}

	resp, err := r.Post(url)

	return newResult(err, resp)
}

func (client *clientImpl) checkURL(s string) (string, error) {
	u, err := url.Parse(s)

	if err != nil {
		return "", err
	}

	u.Path = filepath.Clean(u.Path)

	return u.String(), nil
}

func (client *clientImpl) requestToMap(request interface{}) (map[string]string, error) {
	var params map[string]interface{}

	buff, err := json.Marshal(request)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(buff, &params)

	if err != nil {
		return nil, err
	}

	r := make(map[string]string)

	for k, v := range params {
		r[k] = fmt.Sprintf("%v", v)
	}

	return r, nil
}

func (client *clientImpl) GET(path string, request interface{}, options ...Option) Result {

	params, err := client.requestToMap(request)

	if err != nil {
		return newResult(err, nil)
	}

	r := resty.R().SetQueryParams(params)

	for _, option := range options {
		option(r.RawRequest)
	}

	url, err := client.checkURL(fmt.Sprintf("%s%s", client.url, path))

	if err != nil {
		return newResult(err, nil)
	}

	resp, err := r.Get(url)

	return newResult(err, resp)
}

func (client *clientImpl) DELETE(path string, request interface{}, options ...Option) Result {

	params, err := client.requestToMap(request)

	if err != nil {
		return newResult(err, nil)
	}

	r := resty.R().SetQueryParams(params)

	for _, option := range options {
		option(r.RawRequest)
	}

	url, err := client.checkURL(fmt.Sprintf("%s%s", client.url, path))

	if err != nil {
		return newResult(err, nil)
	}

	resp, err := r.Delete(url)

	return newResult(err, resp)
}
