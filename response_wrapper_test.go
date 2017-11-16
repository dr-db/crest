package crest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func alwaysErr() error {
	return fmt.Errorf("always error")
}

func neverErr() error {
	return nil
}

type errContainer struct {
	err error
}

func (e *errContainer) Error() error {
	return e.err
}

func (e *errContainer) Set(err error) {
	e.err = err
}

type failingReader struct{}

func (f *failingReader) Read(b []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func respWithBody(s string) *http.Response {
	r := &http.Response{}
	r.Body = ioutil.NopCloser(strings.NewReader(s))
	r.Header = make(http.Header)
	r.StatusCode = 200
	return r
}

func TestNewResponseWrapperExistingErr(t *testing.T) {
	body := "some body"
	resp := respWithBody(body)
	ec := &errContainer{}
	rw := newResponseWrapper(resp, alwaysErr, ec.Set)
	rwi, ok := rw.(*responseWrapper)
	require.True(t, ok)
	require.Equal(t, resp, rwi.resp)
	require.Empty(t, rwi.body)
	require.NoError(t, ec.Error())
}

func TestNewResponseWrapperBodyReadErr(t *testing.T) {
	resp := &http.Response{}
	resp.Body = ioutil.NopCloser(&failingReader{})
	ec := &errContainer{}
	rw := newResponseWrapper(resp, neverErr, ec.Set)
	rwi, ok := rw.(*responseWrapper)
	require.True(t, ok)
	require.Equal(t, resp, rwi.resp)
	require.Empty(t, rwi.body)
	require.Error(t, ec.Error())
	require.Contains(t, ec.Error().Error(), "read error")
}

func TestNewResponseWrapperOK(t *testing.T) {
	body := "some body"
	resp := respWithBody(body)
	ec := &errContainer{}
	rw := newResponseWrapper(resp, neverErr, ec.Set)
	rwi, ok := rw.(*responseWrapper)
	require.True(t, ok)
	require.Equal(t, resp, rwi.resp)
	require.Equal(t, body, rwi.body)
	require.NoError(t, ec.Error())
}

func TestBody(t *testing.T) {
	expectedBody := "some body"
	var rw ResponseWrapper
	rw = &responseWrapper{
		body: expectedBody,
	}
	require.Equal(t, expectedBody, rw.Body())
}

func TestExpectBodyContains(t *testing.T) {
	body := "some body\nmore lines\nlast line"
	testCases := []struct {
		needle string
		passes bool
	}{
		{"more", true},
		{"body\nmore", true},
		{"line", true},
		{"nmore", false},
		{"missing", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody(body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectBodyContains(testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "needle = %q", testCase.needle)
		} else {
			require.Error(t, ec.Error(), "needle = %q", testCase.needle)
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectBodyContains("not contained")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectBodyEquals(t *testing.T) {
	body := "some body"
	testCases := []struct {
		needle string
		passes bool
	}{
		{"some", false},
		{"body", false},
		{"some body", true},
		{"missing", false},
		{"", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody(body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectBodyEquals(testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "needle = %q", testCase.needle)
		} else {
			require.Error(t, ec.Error(), "needle = %q", testCase.needle)
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectBodyEquals("different")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectBodyNotContains(t *testing.T) {
	body := "some body\nmore lines\nlast line"
	testCases := []struct {
		needle string
		passes bool
	}{
		{"more", false},
		{"body\nmore", false},
		{"line", false},
		{"nmore", true},
		{"missing", true},
	}
	for _, testCase := range testCases {
		resp := respWithBody(body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectBodyNotContains(testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "needle = %q", testCase.needle)
		} else {
			require.Error(t, ec.Error(), "needle = %q", testCase.needle)
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectBodyNotContains(body)
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectBodyNotEquals(t *testing.T) {
	body := "some body"
	testCases := []struct {
		needle string
		passes bool
	}{
		{"some", true},
		{"body", true},
		{"some body", false},
		{"missing", true},
		{"", true},
	}
	for _, testCase := range testCases {
		resp := respWithBody(body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectBodyNotEquals(testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "needle = %q", testCase.needle)
		} else {
			require.Error(t, ec.Error(), "needle = %q", testCase.needle)
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectBodyNotEquals("some body")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectBodyPasses(t *testing.T) {
	containsWrapper := func(needle string) func(string) bool {
		return func(s string) bool {
			return strings.Contains(s, needle)
		}
	}
	body := "some body\nmore lines\nlast line"
	testCases := []struct {
		needle string
		passes bool
	}{
		{"more", true},
		{"body\nmore", true},
		{"line", true},
		{"nmore", false},
		{"missing", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody(body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectBodyPasses(containsWrapper(testCase.needle))
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "needle = %q", testCase.needle)
		} else {
			require.Error(t, ec.Error(), "needle = %q", testCase.needle)
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectBodyPasses(containsWrapper("not contained"))
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectHeaderContains(t *testing.T) {
	testCases := []struct {
		key    string
		needle string
		passes bool
	}{
		{"Auth", "password", true},
		{"Auth", "pass", true},
		{"Auth", "sword", true},
		{"Auth", "nope", false},
		{"Fake", "", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderContains(testCase.key, testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		} else {
			require.Error(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		}
	}

	resp := respWithBody("")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderContains("missing", "not contained")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderContains("missing", "not contained")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Contains(t, ec.Error().Error(), "no headers")
}

func TestExpectHeaderEquals(t *testing.T) {
	testCases := []struct {
		key    string
		needle string
		passes bool
	}{
		{"Auth", "password", true},
		{"Auth", "pass", false},
		{"Auth", "sword", false},
		{"Auth", "nope", false},
		{"Fake", "", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderEquals(testCase.key, testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		} else {
			require.Error(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		}
	}

	resp := respWithBody("")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderEquals("missing", "not contained")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderEquals("missing", "not contained")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Contains(t, ec.Error().Error(), "no headers")
}

func TestExpectHeaderNotContains(t *testing.T) {
	testCases := []struct {
		key    string
		needle string
		passes bool
	}{
		{"Auth", "password", false},
		{"Auth", "pass", false},
		{"Auth", "sword", false},
		{"Auth", "nope", true},
		{"Fake", "", true},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderNotContains(testCase.key, testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		} else {
			require.Error(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		}
	}

	resp := respWithBody("")
	resp.Header.Add("Auth", "password")
	resp.Header.Add("Other", "header")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderNotContains("Auth", "password")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderNotContains("Auth", "password")
	require.Equal(t, rw, rw2)
	require.NoError(t, ec.Error())
}

func TestExpectHeaderNotEquals(t *testing.T) {
	testCases := []struct {
		key    string
		needle string
		passes bool
	}{
		{"Auth", "password", false},
		{"Auth", "pass", true},
		{"Auth", "sword", true},
		{"Auth", "nope", true},
		{"Fake", "", true},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderNotEquals(testCase.key, testCase.needle)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		} else {
			require.Error(t, ec.Error(), "key = %q, needle = %q", testCase.key, testCase.needle)
		}
	}

	resp := respWithBody("")
	resp.Header.Add("Auth", "password")
	resp.Header.Add("Other", "header")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderNotEquals("Auth", "password")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderNotEquals("Auth", "password")
	require.Equal(t, rw, rw2)
	require.NoError(t, ec.Error())
}

func TestExpectHeaderNotPresent(t *testing.T) {
	testCases := []struct {
		key    string
		passes bool
	}{
		{"Auth", false},
		{"Fake", true},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderNotPresent(testCase.key)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q", testCase.key)
		} else {
			require.Error(t, ec.Error(), "key = %q", testCase.key)
		}
	}

	resp := respWithBody("")
	resp.Header.Add("Auth", "password")
	resp.Header.Add("Other", "header")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderNotPresent("Auth")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderNotPresent("Auth")
	require.Equal(t, rw, rw2)
	require.NoError(t, ec.Error())
}

func TestExpectHeaderPresent(t *testing.T) {
	testCases := []struct {
		key    string
		passes bool
	}{
		{"Auth", true},
		{"Fake", false},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectHeaderPresent(testCase.key)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error(), "key = %q", testCase.key)
		} else {
			require.Error(t, ec.Error(), "key = %q", testCase.key)
		}
	}

	resp := respWithBody("")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectHeaderPresent("missing")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())

	resp = respWithBody("")
	resp.Header = nil
	ec = &errContainer{}
	rw = newResponseWrapper(resp, ec.Error, ec.Set)
	rw2 = rw.ExpectHeaderPresent("missing")
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Contains(t, ec.Error().Error(), "no headers")
}

func TestExpectPasses(t *testing.T) {
	testCases := []struct {
		f      func(*http.Response, string) bool
		passes bool
	}{
		{
			func(resp *http.Response, body string) bool {
				return true
			},
			true,
		},
		{
			func(resp *http.Response, body string) bool {
				return false
			},
			false,
		},
	}
	body := "some body"
	for _, testCase := range testCases {
		resp := respWithBody(body)
		resp.Header.Add("Auth", "password")
		resp.Header.Add("Other", "header")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectPasses(testCase.f)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error())
		} else {
			require.Error(t, ec.Error())
		}
	}

	resp := respWithBody(body)
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectPasses(func(r *http.Response, b string) bool {
		require.Equal(t, resp, r)
		require.Equal(t, body, b)
		return false
	})
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestExpectStatus(t *testing.T) {
	testCases := []struct {
		code   int
		passes bool
	}{
		{200, true},
		{500, false},
	}
	for _, testCase := range testCases {
		resp := respWithBody("")
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		rw2 := rw.ExpectStatus(testCase.code)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error())
		} else {
			require.Error(t, ec.Error())
		}
	}

	resp := respWithBody("")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	rw2 := rw.ExpectStatus(500)
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestParseBody(t *testing.T) {
	type KV struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	testCases := []struct {
		body   string
		passes bool
	}{
		{`{"key": "k", "value": "v"}`, true},
		{`{"value": "v", "key": "k"}`, true},
		{`not JSON`, false},
	}
	for _, testCase := range testCases {
		resp := respWithBody(testCase.body)
		ec := &errContainer{}
		rw := newResponseWrapper(resp, neverErr, ec.Set)
		var actual KV
		rw2 := rw.ParseBody(&actual)
		require.Equal(t, rw, rw2)
		if testCase.passes {
			require.NoError(t, ec.Error())
			expected := KV{
				Key:   "k",
				Value: "v",
			}
			require.Equal(t, expected, actual)
		} else {
			require.Error(t, ec.Error())
		}
	}

	resp := respWithBody("not JSON")
	existingError := fmt.Errorf("existing error")
	ec := &errContainer{}
	rw := newResponseWrapper(resp, ec.Error, ec.Set)
	ec.Set(existingError)
	var kv KV
	rw2 := rw.ParseBody(&kv)
	require.Equal(t, rw, rw2)
	require.Error(t, ec.Error())
	require.Equal(t, existingError, ec.Error())
}

func TestNopResponseWrapper(t *testing.T) {
	var n nopResponseWrapper
	require.Equal(t, "", n.Body())
	require.Equal(t, n, n.ExpectBodyContains(""))
	require.Equal(t, n, n.ExpectBodyEquals(""))
	require.Equal(t, n, n.ExpectBodyNotContains(""))
	require.Equal(t, n, n.ExpectBodyNotEquals(""))
	require.Equal(t, n, n.ExpectBodyPasses(func(string) bool { return true }))
	require.Equal(t, n, n.ExpectHeaderContains("", ""))
	require.Equal(t, n, n.ExpectHeaderEquals("", ""))
	require.Equal(t, n, n.ExpectHeaderNotContains("", ""))
	require.Equal(t, n, n.ExpectHeaderNotEquals("", ""))
	require.Equal(t, n, n.ExpectHeaderNotPresent(""))
	require.Equal(t, n, n.ExpectHeaderPresent(""))
	require.Equal(t, n, n.ExpectPasses(func(resp *http.Response, body string) bool { return true }))
	require.Equal(t, n, n.ExpectStatus(0))
	require.Equal(t, n, n.ParseBody(""))
}
