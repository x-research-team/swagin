package router

import (
	"net/http"
	_ "unsafe"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
)

var Query = queryBinding{}

func CookiesParser(c *gin.Context, model any) error {
	params := make(map[string][]string)
	for _, cookie := range c.Request.Cookies() {
		params[cookie.Name] = append(params[cookie.Name], cookie.Value)
	}
	return copier.Copy(model, params)
}

type queryBinding struct{}

func (queryBinding) Name() string {
	return "query"
}

func (queryBinding) Bind(req *http.Request, obj any) error {
	values := req.URL.Query()
	m := make(map[string]string)
	for k, v := range values {
		m[k] = v[0]
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "query",
		Result:  obj,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(m)
}

var Header = headerBinding{}

type headerBinding struct{}

func (headerBinding) Name() string {
	return "header"
}

func (headerBinding) Bind(req *http.Request, obj any) error {
	values := req.Header
	m := make(map[string]string)
	for k, v := range values {
		m[k] = v[0]
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "header",
		Result:  obj,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(m)
}
