package restclient

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func handleResult(resp *resty.Response, name string, obj interface{}) error {
	result := make(map[string]interface{})

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return fmt.Errorf("unmarshal result err %s\n%s", err, string(resp.Body()))
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("http code(%s) code(%v) errmsg %s", resp.Status(), result["code"], result["msg"])
	}

	data, ok := result[name]

	if !ok {
		return fmt.Errorf("unknown return value %s\n%s", name, string(resp.Body()))
	}

	buff, err := json.Marshal(data)

	if err != nil {
		return fmt.Errorf("unmarshal result(%s) err %s\n%s", name, err, string(resp.Body()))
	}

	if err := json.Unmarshal(buff, obj); err != nil {
		return fmt.Errorf("unmarshal result(%s) err %s\n%s", name, err, string(buff))
	}

	return nil
}

func (client *clientImpl) POST(path string, request interface{}, options ...Option) Result {

	r := resty.R().SetBody(request) //.Post(fmt.Sprintf("%s/%s", client.url, path))

	for _, option := range options {
		option(r.RawRequest)
	}

	resp, err := r.Post(fmt.Sprintf("%s/%s", client.url, path))

	return newResult(err, resp)
}

func (client *clientImpl) GET(path string, request interface{}, options ...Option) Result {

	var params map[string]string

	buff, err := json.Marshal(request)

	if err != nil {
		return newResult(err, nil)
	}

	err = json.Unmarshal(buff, &params)

	if err != nil {
		return newResult(err, nil)
	}

	r := resty.R().SetQueryParams(params)

	for _, option := range options {
		option(r.RawRequest)
	}

	resp, err := r.Get(fmt.Sprintf("%s/%s", client.url, path))

	return newResult(err, resp)
}

func (client *clientImpl) DELETE(path string, request interface{}, options ...Option) Result {

	var params map[string]string

	buff, err := json.Marshal(request)

	if err != nil {
		return newResult(err, nil)
	}

	err = json.Unmarshal(buff, &params)

	if err != nil {
		return newResult(err, nil)
	}

	r := resty.R().SetQueryParams(params)

	for _, option := range options {
		option(r.RawRequest)
	}

	resp, err := r.Delete(fmt.Sprintf("%s/%s", client.url, path))

	return newResult(err, resp)
}
