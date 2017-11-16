package crest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Client interface {
	NoBasicAuth() Client
	UseBasicAuth(string, string) Client
	UseCookies(bool) Client
	WithHeader(key, value string) Client
	WithTimeout(time.Duration) Client

	Error() error
	Clone() Client

	Delete(path string) ResponseWrapper
	Get(path string) ResponseWrapper
	Patch(path string, body interface{}) ResponseWrapper
	Post(path string, body interface{}) ResponseWrapper
	Put(path string, body interface{}) ResponseWrapper
	PatchNoBody(path string) ResponseWrapper
	PostNoBody(path string) ResponseWrapper
	PutNoBody(path string) ResponseWrapper
	PatchString(path string, body string) ResponseWrapper
	PostString(path string, body string) ResponseWrapper
	PutString(path string, body string) ResponseWrapper
	PatchBytes(path string, body []byte) ResponseWrapper
	PostBytes(path string, body []byte) ResponseWrapper
	PutBytes(path string, body []byte) ResponseWrapper
	PostForm(path string, body url.Values) ResponseWrapper
}

type client struct {
	baseURL    string
	httpClient *http.Client

	err       error
	errGetter func() error
	errSetter func(error)
	errLock   sync.RWMutex

	useBasicAuth  bool
	basicAuthUser string
	basicAuthPass string
	useCookies    bool
	headers       http.Header
	timeout       time.Duration
}

func NewClient(url string) Client {
	return NewCustomClient(url, &http.Client{})
}

func NewCustomClient(url string, httpClient *http.Client) Client {
	cl := &client{
		baseURL:    url,
		httpClient: httpClient,
	}
	cl.errGetter = func() error {
		cl.errLock.RLock()
		defer cl.errLock.RUnlock()

		return cl.err
	}
	cl.errSetter = func(err error) {
		cl.errLock.Lock()
		defer cl.errLock.Unlock()

		cl.err = err
	}
	return cl
}

func (c *client) NoBasicAuth() Client {
	if c.errGetter() != nil {
		return c
	}
	c.useBasicAuth = false
	c.basicAuthUser = ""
	c.basicAuthPass = ""
	return c
}

func (c *client) UseBasicAuth(user, pass string) Client {
	if c.errGetter() != nil {
		return c
	}
	c.useBasicAuth = true
	c.basicAuthUser = user
	c.basicAuthPass = pass
	return c
}

func (c *client) UseCookies(use bool) Client {
	if c.errGetter() != nil {
		return c
	}
	if !use {
		c.httpClient.Jar = nil
		return c
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		c.errSetter(errors.Wrap(err, "creating cookie jar"))
		return c
	}
	c.httpClient.Jar = jar
	return c
}

func (c *client) WithHeader(key, value string) Client {
	if c.errGetter() != nil {
		return c
	}
	if c.headers == nil {
		c.headers = make(http.Header)
	}
	c.headers.Add(key, value)
	return c
}

func (c *client) WithTimeout(timeout time.Duration) Client {
	if c.errGetter() != nil {
		return c
	}
	c.timeout = timeout
	return c
}

func (c *client) Error() error {
	return c.errGetter()
}

func (c *client) Clone() Client {
	if c.errGetter() != nil {
		return c
	}
	cloned := *c
	cloned.headers = make(http.Header)
	for key, vals := range c.headers {
		for _, val := range vals {
			cloned.headers.Add(key, val)
		}
	}
	return &cloned
}

func (c *client) buildPath(path string) string {
	return c.baseURL + "/" + strings.TrimPrefix(path, "/")
}

func (c *client) buildReq(method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, c.buildPath(path), body)
	if err != nil {
		c.errSetter(errors.Wrap(err, "creating request"))
		return nil
	}
	return c.populateReq(req)
}

func (c *client) doReq(method, path string, body io.Reader) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	req := c.buildReq(method, path, body)
	return c.do(req)
}

func (c *client) doReqJSON(method, path string, body interface{}) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	bs, err := json.Marshal(body)
	if err != nil {
		c.errSetter(errors.Wrap(err, "marshalling JSON body"))
		return &nopResponseWrapper{}
	}
	return c.doReq(method, path, bytes.NewBuffer(bs))
}

func (c *client) doReqString(method, path string, body string) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	return c.doReq(method, path, bytes.NewBufferString(body))
}

func (c *client) doReqBytes(method, path string, body []byte) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	return c.doReq(method, path, bytes.NewBuffer(body))
}

func (c *client) doReqNoBody(method, path string) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	return c.doReq(method, path, nil)
}

func (c *client) doReqForm(method, path string, body url.Values) ResponseWrapper {
	if c.errGetter() != nil {
		return &nopResponseWrapper{}
	}
	req := c.buildReq(method, path, bytes.NewBufferString(body.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

func (c *client) populateReq(req *http.Request) *http.Request {
	if c.useBasicAuth {
		req.SetBasicAuth(c.basicAuthUser, c.basicAuthPass)
	}
	for key, vals := range c.headers {
		for _, val := range vals {
			req.Header.Add(key, val)
		}
	}
	if c.timeout > 0 {
		ctx, _ := context.WithTimeout(context.Background(), c.timeout)
		req = req.WithContext(ctx)
	}
	return req
}

func (c *client) do(req *http.Request) ResponseWrapper {
	if c.errGetter() != nil {
		return newResponseWrapper(nil, c.Error, c.errSetter)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.errSetter(errors.Wrap(err, "doing request"))
	}
	return newResponseWrapper(resp, c.Error, func(err error) {
		c.errSetter(errors.Wrapf(err, "doing a %v request to URL %q", req.Method, req.URL.String()))
	})
}

func (c *client) Delete(path string) ResponseWrapper {
	return c.doReqNoBody(http.MethodDelete, path)
}

func (c *client) Get(path string) ResponseWrapper {
	return c.doReqNoBody(http.MethodGet, path)
}

func (c *client) Patch(path string, body interface{}) ResponseWrapper {
	return c.doReqJSON(http.MethodPatch, path, body)
}

func (c *client) Post(path string, body interface{}) ResponseWrapper {
	return c.doReqJSON(http.MethodPost, path, body)
}

func (c *client) Put(path string, body interface{}) ResponseWrapper {
	return c.doReqJSON(http.MethodPut, path, body)
}

func (c *client) PatchNoBody(path string) ResponseWrapper {
	return c.doReqNoBody(http.MethodPatch, path)
}

func (c *client) PostNoBody(path string) ResponseWrapper {
	return c.doReqNoBody(http.MethodPost, path)
}

func (c *client) PutNoBody(path string) ResponseWrapper {
	return c.doReqNoBody(http.MethodPut, path)
}

func (c *client) PatchString(path string, body string) ResponseWrapper {
	return c.doReqString(http.MethodPatch, path, body)
}

func (c *client) PostString(path string, body string) ResponseWrapper {
	return c.doReqString(http.MethodPost, path, body)
}

func (c *client) PutString(path string, body string) ResponseWrapper {
	return c.doReqString(http.MethodPut, path, body)
}

func (c *client) PatchBytes(path string, body []byte) ResponseWrapper {
	return c.doReqBytes(http.MethodPatch, path, body)
}

func (c *client) PostBytes(path string, body []byte) ResponseWrapper {
	return c.doReqBytes(http.MethodPost, path, body)
}

func (c *client) PutBytes(path string, body []byte) ResponseWrapper {
	return c.doReqBytes(http.MethodPut, path, body)
}

func (c *client) PostForm(path string, body url.Values) ResponseWrapper {
	return c.doReqForm(http.MethodPost, path, body)
}
