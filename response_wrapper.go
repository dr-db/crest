package crest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type ResponseWrapper interface {
	Body() string
	ExpectBodyContains(string) ResponseWrapper
	ExpectBodyEquals(string) ResponseWrapper
	ExpectBodyNotContains(string) ResponseWrapper
	ExpectBodyNotEquals(string) ResponseWrapper
	ExpectBodyPasses(func(string) bool) ResponseWrapper
	ExpectHeaderContains(key, value string) ResponseWrapper
	ExpectHeaderEquals(key, value string) ResponseWrapper
	ExpectHeaderNotContains(key, value string) ResponseWrapper
	ExpectHeaderNotEquals(key, value string) ResponseWrapper
	ExpectHeaderNotPresent(key string) ResponseWrapper
	ExpectHeaderPresent(key string) ResponseWrapper
	ExpectPasses(func(resp *http.Response, body string) bool) ResponseWrapper
	ExpectStatus(int) ResponseWrapper
	ParseBody(interface{}) ResponseWrapper
}

func newResponseWrapper(resp *http.Response, errChecker func() error, errSetter func(error)) ResponseWrapper {
	r := &responseWrapper{
		error:    errChecker,
		resp:     resp,
		setError: errSetter,
	}

	if errChecker() != nil {
		return r
	}

	if bs, err := ioutil.ReadAll(r.resp.Body); err != nil {
		r.setError(errors.Wrap(err, "reading response body"))
	} else {
		r.body = string(bs)
	}

	return r
}

type responseWrapper struct {
	error    func() error
	setError func(error)

	resp *http.Response
	body string
}

func (r *responseWrapper) Body() string {
	return r.body
}

func (r *responseWrapper) ExpectBodyContains(needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if !strings.Contains(r.body, needle) {
		r.setError(fmt.Errorf("expected body to contain %q but it did not", needle))
	}
	return r
}

func (r *responseWrapper) ExpectBodyEquals(value string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.body != value {
		r.setError(fmt.Errorf("expected body to be %q but it was not", value))
	}
	return r
}

func (r *responseWrapper) ExpectBodyNotContains(needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if strings.Contains(r.body, needle) {
		r.setError(fmt.Errorf("expected body to not contain %q but it does", needle))
	}
	return r
}

func (r *responseWrapper) ExpectBodyNotEquals(value string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.body == value {
		r.setError(fmt.Errorf("expected body not to be %q but it was", value))
	}
	return r
}

func (r *responseWrapper) ExpectBodyPasses(f func(string) bool) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if !f(r.body) {
		r.setError(fmt.Errorf("expected function to pass, but it did not"))
	}
	return r
}

func (r *responseWrapper) ExpectHeaderContains(key, needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.resp.Header == nil {
		r.setError(fmt.Errorf("expected a header %q containing %q, but there are no headers", key, needle))
		return r
	}

	found := false
	for _, value := range r.resp.Header[key] {
		if strings.Contains(value, needle) {
			found = true
			break
		}
	}
	if !found {
		r.setError(fmt.Errorf("expected a header %q containing %q, but it did not", key, needle))
	}

	return r
}

func (r *responseWrapper) ExpectHeaderEquals(key, needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.resp.Header == nil {
		r.setError(fmt.Errorf("expected a header %q containing %q, but there are no headers", key, needle))
		return r
	}

	found := false
	for _, value := range r.resp.Header[key] {
		if value == needle {
			found = true
			break
		}
	}
	if !found {
		r.setError(fmt.Errorf("expected a header %q containing %q, but it did not", key, needle))
	}

	return r
}

func (r *responseWrapper) ExpectHeaderNotContains(key, needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.resp.Header == nil {
		return r
	}

	found := false
	for _, value := range r.resp.Header[key] {
		if strings.Contains(value, needle) {
			found = true
			break
		}
	}
	if found {
		r.setError(fmt.Errorf("expected a header %q to not contain %q, but it does", key, needle))
	}

	return r
}

func (r *responseWrapper) ExpectHeaderNotEquals(key, needle string) ResponseWrapper {
	if r.error() != nil {
		return r
	}

	found := false
	for _, value := range r.resp.Header[key] {
		if value == needle {
			found = true
			break
		}
	}
	if found {
		r.setError(fmt.Errorf("expected a header %q to not be %q, but it is", key, needle))
	}

	return r
}

func (r *responseWrapper) ExpectHeaderNotPresent(key string) ResponseWrapper {
	if r.error() != nil {
		return r
	}

	if len(r.resp.Header[key]) > 0 {
		r.setError(fmt.Errorf("expected a header %q not to be present, but it was", key))
	}

	return r
}

func (r *responseWrapper) ExpectHeaderPresent(key string) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.resp.Header == nil {
		r.setError(fmt.Errorf("expected a header %q, but there are no headers", key))
		return r
	}

	if len(r.resp.Header[key]) == 0 {
		r.setError(fmt.Errorf("expected a header %q to be present, but it was not", key))
	}

	return r
}

func (r *responseWrapper) ExpectPasses(f func(*http.Response, string) bool) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if !f(r.resp, r.body) {
		r.setError(fmt.Errorf("expected function to pass, but it did not"))
	}

	return r
}

func (r *responseWrapper) ExpectStatus(code int) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if r.resp.StatusCode != code {
		r.setError(fmt.Errorf("expected status code %d but got %d", code, r.resp.StatusCode))
	}

	return r
}

func (r *responseWrapper) ParseBody(v interface{}) ResponseWrapper {
	if r.error() != nil {
		return r
	}
	if err := json.Unmarshal([]byte(r.body), v); err != nil {
		r.setError(fmt.Errorf("unmarshalling body: %v", err))
	}

	return r
}

type nopResponseWrapper struct{}

func (n nopResponseWrapper) Body() string {
	return ""
}

func (n nopResponseWrapper) ExpectBodyContains(string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectBodyEquals(string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectBodyNotContains(string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectBodyNotEquals(string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectBodyPasses(func(string) bool) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderContains(key, value string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderEquals(key, value string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderNotContains(key, value string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderNotEquals(key, value string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderNotPresent(key string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectHeaderPresent(key string) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectPasses(func(resp *http.Response, body string) bool) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ExpectStatus(int) ResponseWrapper {
	return n
}

func (n nopResponseWrapper) ParseBody(interface{}) ResponseWrapper {
	return n
}
