package main

import (
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/x-research-team/swagin/swagger"
)

type option func(*swagger.Swagger)

func New(opts ...option) *swagger.Swagger {
	return swagger.New("SwaGin", "Swagger + Gin = SwaGin", "0.1.0",
		swagger.License(&openapi3.License{
			Name: "Apache License 2.0",
			URL:  "https://github.com/x-research-team/swagin/blob/dev/LICENSE",
		}),
		swagger.Contact(&openapi3.Contact{
			Name:  "long2ice",
			URL:   "https://github.com/x-research-team/swagin",
			Email: "long2ice@gmail.com",
		}),
		swagger.TermsOfService("https://github.com/long2ice"),
	)
}
