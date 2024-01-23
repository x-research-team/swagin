package swagger

import (
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/x-research-team/swagin/router"
)

type Option func(swagger *Swagger)

func Routers(routers map[string]map[string]*router.Router) Option {
	return func(swagger *Swagger) {
		swagger.Routers = routers
	}
}
func DocsUrl(url string) Option {
	return func(swagger *Swagger) {
		swagger.DocsUrl = url
	}
}
func RedocUrl(url string) Option {
	return func(swagger *Swagger) {
		swagger.RedocUrl = url
	}
}
func Title(title string) Option {
	return func(swagger *Swagger) {
		swagger.Title = title
	}
}
func Description(description string) Option {
	return func(swagger *Swagger) {
		swagger.Description = description
	}
}
func Version(version string) Option {
	return func(swagger *Swagger) {
		swagger.Version = version
	}
}
func OpenAPIUrl(url string) Option {
	return func(swagger *Swagger) {
		swagger.OpenAPIUrl = url
	}
}
func Servers(servers openapi3.Servers) Option {
	return func(swagger *Swagger) {
		swagger.Servers = servers
	}
}
func TermsOfService(tos string) Option {
	return func(swagger *Swagger) {
		swagger.TermsOfService = tos
	}
}
func Contact(contact *openapi3.Contact) Option {
	return func(swagger *Swagger) {
		swagger.Contact = contact
	}
}
func License(lic *openapi3.License) Option {
	return func(swagger *Swagger) {
		swagger.License = lic
	}
}
func SwaggerOptions(options map[string]any) Option {
	return func(swagger *Swagger) {
		swagger.SwaggerOptions = options
	}
}
func RedocOptions(options map[string]any) Option {
	return func(swagger *Swagger) {
		swagger.RedocOptions = options
	}
}
