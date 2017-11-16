package crest

import "log"

func main() {
	baseURL := "https://httpbin.org/"

	cr := NewClient(baseURL)

	ui := cr.Clone().
		WithHeader("h1", "v1").
		WithHeader("h2", "v2").
		UseCookies(true)
	api := cr.Clone().
		WithHeader("api-key", "12345").
		UseCookies(false)

	api.Post("/path", "JSON body or object").
		ExpectStatus(201) // Default is 200
	ui.Get("/path/check").
		ExpectBodyContains("new key")

	api.Get("/path").
		ExpectStatus(400).
		ExpectBodyNotContains("missing").
		ExpectBodyPasses(func(body string) bool {
			return true
		}).
		Body()

	var j struct {
		Field string
	}
	api.Get("/path").
		ParseBody(&j)

	if err := cr.Error(); err != nil {
		log.Fatal(err)
	}
}
