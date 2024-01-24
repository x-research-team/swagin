package swagger

import (
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/goccy/go-reflect"

	"github.com/fatih/structtag"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/yaml.v2"

	"github.com/x-research-team/swagin/router"
	"github.com/x-research-team/swagin/security"
)

const (
	DEFAULT     = "default"
	BINDING     = "binding"
	DESCRIPTION = "description"
	QUERY       = "query"
	FORM        = "form"
	URI         = "uri"
	HEADER      = "header"
	COOKIE      = "cookie"
	JSON        = "json"
	RULE        = "rule"
	EXAMPLE     = "example"
	FORMAT      = "format"
)

type Swagger struct {
	Title          string
	Description    string
	Version        string
	DocsUrl        string
	RedocUrl       string
	OpenAPIUrl     string
	Routers        map[string]map[string]*router.Router
	Servers        openapi3.Servers
	TermsOfService string
	Contact        *openapi3.Contact
	License        *openapi3.License
	OpenAPI        *openapi3.T
	SwaggerOptions map[string]any
	RedocOptions   map[string]any
}

func New(title, description, version string, options ...Option) *Swagger {
	swagger := &Swagger{Title: title, Description: description, Version: version, DocsUrl: "/docs", RedocUrl: "/redoc", OpenAPIUrl: "/openapi.json"}
	for _, option := range options {
		option(swagger)
	}
	return swagger
}
func (swagger *Swagger) getSecurityRequirements(securities []security.ISecurity) *openapi3.SecurityRequirements {
	securityRequirements := openapi3.NewSecurityRequirements()
	for _, s := range securities {
		provide := s.Provider()
		swagger.OpenAPI.Components.SecuritySchemes[provide] = &openapi3.SecuritySchemeRef{
			Value: s.Scheme(),
		}
		securityRequirements.With(openapi3.NewSecurityRequirement().Authenticate(provide))
	}
	return securityRequirements
}
func (swagger *Swagger) getSchemaByType(t any, request bool) *openapi3.Schema {
	var schema *openapi3.Schema
	var m = float64(0)
	switch t.(type) {
	case int, int8, int16:
		schema = openapi3.NewIntegerSchema()
	case uint, uint8, uint16:
		schema = openapi3.NewIntegerSchema()
		schema.Min = &m
	case int32:
		schema = openapi3.NewInt32Schema()
	case uint32:
		schema = openapi3.NewInt32Schema()
		schema.Min = &m
	case int64:
		schema = openapi3.NewInt64Schema()
	case uint64:
		schema = openapi3.NewInt64Schema()
		schema.Min = &m
	case string:
		schema = openapi3.NewStringSchema()
	case time.Time:
		schema = openapi3.NewDateTimeSchema()
	case float32, float64:
		schema = openapi3.NewFloat64Schema()
	case bool:
		schema = openapi3.NewBoolSchema()
	case []byte:
		schema = openapi3.NewBytesSchema()
	case *multipart.FileHeader:
		schema = openapi3.NewStringSchema()
		schema.Format = "binary"
	case []*multipart.FileHeader:
		schema = openapi3.NewArraySchema()
		schema.Items = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:   "string",
				Format: "binary",
			},
		}
	default:
		if request {
			schema = swagger.getRequestSchemaByModel(t)
		} else {
			schema = swagger.getResponseSchemaByModel(t)
		}
	}
	return schema
}
func (swagger *Swagger) getRequestSchemaByModel(model any) *openapi3.Schema {
	type_ := reflect.TypeOf(model)
	value_ := reflect.ValueOf(model)
	schema := openapi3.NewObjectSchema()
	if type_.Kind() == reflect.Ptr {
		type_ = type_.Elem()
	}
	if value_.Kind() == reflect.Ptr {
		if value_.IsNil() {
			value_ = reflect.New(value_.Type().Elem())
		}
		value_ = value_.Elem()
	}
	if type_.Kind() == reflect.Struct {
		for i := 0; i < type_.NumField(); i++ {
			field := type_.Field(i)
			value := value_.Field(i)
			tags, err := structtag.Parse(string(field.Tag))
			if err != nil {
				panic(err)
			}
			tag, err := tags.Get(JSON)
			if err != nil {
				continue
			}

			if tag.Name == "-" {
				continue
			}

			fieldSchema := swagger.getSchemaByType(value.Interface(), true)
			descriptionTag, err := tags.Get(DESCRIPTION)
			if err == nil {
				fieldSchema.Description = descriptionTag.Name
			}
			bindingTag, err := tags.Get(BINDING)
			if err == nil {
				if bindingTag.Name == "required" {
					schema.Required = append(schema.Required, tag.Name)
				}
			}
			defaultTag, err := tags.Get(DEFAULT)
			if err == nil {
				fieldSchema.Default = defaultTag.Name
			}
			exampleTag, err := tags.Get(EXAMPLE)
			if err == nil {
				fieldSchema.Example = exampleTag.Name
			}
			ruleTag, err := tags.Get(RULE)
			if err == nil {
				pattern := params("regexp", ruleTag.Name)
				if len(pattern) == 0 {
					for _, o := range ruleTag.Options {
						pattern = params("regexp", o)
						if len(pattern) == 1 {
							fieldSchema.Pattern = pattern[0]
							break
						}
					}
				}
				isNonNilExists := isParamExist("nonnil", ruleTag.Name)
				isNonZeroExists := isParamExist("nonzero", ruleTag.Name)
				if !isNonNilExists || !isNonZeroExists {
					for _, o := range ruleTag.Options {
						isNonNilExists = isParamExist("nonnil", o)
						isNonZeroExists = isParamExist("nonzero", o)
						if isNonNilExists || isNonZeroExists {
							fieldSchema.Required = append(fieldSchema.Required, "required")
							break
						}
					}
				}
			}
			formatTag, err := tags.Get(FORMAT)
			if err == nil {
				fieldSchema.Format = formatTag.Name
			}
			schema.Properties[tag.Name] = openapi3.NewSchemaRef("", fieldSchema)
		}
	} else if type_.Kind() == reflect.Slice {
		schema = openapi3.NewArraySchema()
		if type_.Elem().Kind() == reflect.Ptr {
			schema.Items = &openapi3.SchemaRef{Value: swagger.getRequestSchemaByModel(reflect.New(type_.Elem().Elem()).Elem().Interface())}
		} else {
			schema.Items = &openapi3.SchemaRef{Value: swagger.getRequestSchemaByModel(reflect.New(type_.Elem()).Elem().Interface())}
		}
	} else if type_.Kind() == reflect.Map {
		schema = openapi3.NewObjectSchema()
		schema.Items = &openapi3.SchemaRef{Value: swagger.getRequestSchemaByModel(reflect.New(type_.Elem()).Elem().Interface())}
	} else {
		schema = swagger.getSchemaByType(value_.Interface(), true)
	}
	return schema
}
func (swagger *Swagger) getRequestBodyByModel(model any, contentType string) *openapi3.RequestBodyRef {
	body := &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody(),
	}
	if model == nil {
		return body
	}
	schema := swagger.getRequestSchemaByModel(model)
	body.Value.Required = true
	if contentType == "" {
		contentType = binding.MIMEJSON
	}
	body.Value.Content = openapi3.NewContentWithSchema(schema, []string{contentType})
	return body
}
func (swagger *Swagger) getResponseSchemaByModel(model any) *openapi3.Schema {
	type_ := reflect.TypeOf(model)
	value_ := reflect.ValueOf(model)
	if type_.Kind() == reflect.Ptr {
		type_ = type_.Elem()
	}
	if value_.Kind() == reflect.Ptr {
		value_ = value_.Elem()
	}
	schema := openapi3.NewObjectSchema()
	if type_.Kind() == reflect.Struct {
		for i := 0; i < type_.NumField(); i++ {
			field := reflect.ToReflectType(type_).Field(i)
			if field.IsExported() && value_.IsValid() {
				value := value_.Field(i)
				if value.Kind() == reflect.Ptr {
					if value.IsNil() {
						value = reflect.New(value.Type().Elem())
					}
					value = value.Elem()
				}
				fieldSchema := swagger.getSchemaByType(value.Interface(), false)
				tags, err := structtag.Parse(string(field.Tag))
				if err != nil {
					panic(err)
				}
				tag, err := tags.Get(JSON)
				if err != nil {
					continue
				}
				if tag.Name == "-" {
					continue
				}
				bindingTag, err := tags.Get(BINDING)
				if err == nil && bindingTag.Name == "required" {
					schema.Required = append(schema.Required, tag.Name)
				}
				descriptionTag, err := tags.Get(DESCRIPTION)
				if err == nil {
					fieldSchema.Description = descriptionTag.Name
				}
				defaultTag, err := tags.Get(DEFAULT)
				if err == nil {
					fieldSchema.Default = defaultTag.Name
				}
				exampleTag, err := tags.Get(EXAMPLE)
				if err == nil {
					fieldSchema.Example = exampleTag.Name
				}
				ruleTag, err := tags.Get(RULE)
				if err == nil {
					pattern := params("regexp", ruleTag.Name)
					if len(pattern) == 0 {
						for _, o := range ruleTag.Options {
							pattern = params("regexp", o)
							if len(pattern) == 1 {
								fieldSchema.Pattern = pattern[0]
								break
							}
						}
					}
					isNonNilExists := isParamExist("nonnil", ruleTag.Name)
					isNonZeroExists := isParamExist("nonzero", ruleTag.Name)
					if !isNonNilExists || !isNonZeroExists {
						for _, o := range ruleTag.Options {
							isNonNilExists = isParamExist("nonnil", o)
							isNonZeroExists = isParamExist("nonzero", o)
							if isNonNilExists || isNonZeroExists {
								fieldSchema.Required = append(fieldSchema.Required, "required")
								break
							}
						}
					}
				}
				formatTag, err := tags.Get(FORMAT)
				if err == nil {
					fieldSchema.Format = formatTag.Name
				}
				schema.Properties[tag.Name] = openapi3.NewSchemaRef("", fieldSchema)
			}
		}
	} else if type_.Kind() == reflect.Slice {
		schema = openapi3.NewArraySchema()
		if type_.Elem().Kind() == reflect.Ptr {
			schema.Items = &openapi3.SchemaRef{Value: swagger.getResponseSchemaByModel(reflect.New(type_.Elem().Elem()).Elem().Interface())}
		} else {
			schema.Items = &openapi3.SchemaRef{Value: swagger.getResponseSchemaByModel(reflect.New(type_.Elem()).Elem().Interface())}
		}
	} else if type_.Kind() == reflect.Map {
		schema = openapi3.NewObjectSchema()
		schema.Items = &openapi3.SchemaRef{Value: swagger.getResponseSchemaByModel(reflect.New(type_.Elem()).Elem().Interface())}
	} else {
		schema = swagger.getSchemaByType(model, false)
	}
	return schema
}
func (swagger *Swagger) getResponses(response router.Response, contentType string) *openapi3.Responses {
	ret := openapi3.NewResponses()
	for k, v := range response {
		schema := swagger.getResponseSchemaByModel(v.Model)
		var content openapi3.Content
		if contentType == "" || contentType == binding.MIMEJSON {
			content = openapi3.NewContentWithJSONSchema(schema)
		} else {
			content = openapi3.NewContentWithSchema(schema, []string{contentType})
		}
		description := v.Description
		ret.Set(k, &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &description,
				Content:     content,
				Headers:     v.Headers,
			},
		})
	}
	return ret
}

func params(key string, name string) []string {
	for _, value := range strings.Split(name, ",") {
		if strings.HasPrefix(value, key+"=") {
			return strings.Split(value[len(key+"="):], ",")
		}
	}
	return nil
}

func isParamExist(key string, name string) bool {
	for _, value := range strings.Split(name, ",") {
		if value == key {
			return true
		}
	}
	return false
}

func (swagger *Swagger) getParamTypeByModel(model any, param string) openapi3.Parameters {
	parameters := openapi3.NewParameters()
	if model == nil {
		return parameters
	}
	type_ := reflect.TypeOf(model)
	if type_.Kind() == reflect.Ptr {
		type_ = type_.Elem()
	}
	value_ := reflect.ValueOf(model)
	if value_.Kind() == reflect.Ptr {
		value_ = value_.Elem()
	}
	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		value := value_.Field(i)
		tags, err := structtag.Parse(string(field.Tag))
		if err != nil {
			panic(err)
		}
		parameter := &openapi3.Parameter{}
		paramTag, err := tags.Get(param)
		if err == nil {
			switch param {
			case QUERY:
				parameter.In = openapi3.ParameterInQuery
			case URI:
				parameter.In = openapi3.ParameterInPath
			case COOKIE:
				parameter.In = openapi3.ParameterInCookie
			case HEADER:
				parameter.In = openapi3.ParameterInHeader
			}
			parameter.Name = paramTag.Name
		}

		if parameter.In == "" {
			continue
		}
		descriptionTag, err := tags.Get(DESCRIPTION)
		if err == nil {
			parameter.Description = descriptionTag.Name
		}
		bindingTag, err := tags.Get(BINDING)
		if err == nil {
			parameter.Required = bindingTag.Name == "required"
		}
		exampleTag, err := tags.Get(EXAMPLE)
		if err == nil {
			parameter.Example = exampleTag.Name
		}
		defaultTag, err := tags.Get(DEFAULT)
		schema := swagger.getSchemaByType(value.Interface(), true)
		if err == nil {
			schema.Default = defaultTag.Name
		}
		parameter.Schema = &openapi3.SchemaRef{
			Value: schema,
		}
		ruleTag, err := tags.Get(RULE)
		if err == nil {
			pattern := params("regexp", ruleTag.Name)
			if len(pattern) == 0 {
				for _, o := range ruleTag.Options {
					pattern = params("regexp", o)
					if len(pattern) == 1 {
						schema.Pattern = pattern[0]
						break
					}
				}
			}
			isNonNilExists := isParamExist("nonnil", ruleTag.Name)
			isNonZeroExists := isParamExist("nonzero", ruleTag.Name)
			if !isNonNilExists || !isNonZeroExists {
				for _, o := range ruleTag.Options {
					isNonNilExists = isParamExist("nonnil", o)
					isNonZeroExists = isParamExist("nonzero", o)
					if isNonNilExists || isNonZeroExists {
						parameter.Required = true
						break
					}
				}
			}
		}
		formatTag, err := tags.Get(FORMAT)
		if err == nil {
			schema.Format = formatTag.Name
		}
		parameters = append(parameters, &openapi3.ParameterRef{
			Value: parameter,
		})
	}
	return parameters
}

func (swagger *Swagger) getParametersByModel(model any) openapi3.Parameters {
	parameters := openapi3.NewParameters()
	if model == nil {
		return parameters
	}
	value_ := reflect.ValueOf(model)
	if value_.Kind() == reflect.Ptr {
		value_ = value_.Elem()
	}
	uriField := value_.FieldByName("URI")
	if uriField.IsValid() {
		parameters = append(parameters, swagger.getParamTypeByModel(uriField.Interface(), URI)...)
	}
	queryField := value_.FieldByName("Query")
	if queryField.IsValid() {
		parameters = append(parameters, swagger.getParamTypeByModel(queryField.Interface(), QUERY)...)
	}
	cookieField := value_.FieldByName("Cookie")
	if cookieField.IsValid() {
		parameters = append(parameters, swagger.getParamTypeByModel(cookieField.Interface(), COOKIE)...)
	}
	headerField := value_.FieldByName("Header")
	if headerField.IsValid() {
		parameters = append(parameters, swagger.getParamTypeByModel(headerField.Interface(), HEADER)...)
	}
	return parameters
}

// /:id -> /{id}
func (swagger *Swagger) fixPath(path string) string {
	reg := regexp.MustCompile(":([a-zA-Z0-9_]+)")
	return reg.ReplaceAllString(path, "{$1}")
}

func (swagger *Swagger) hasSchemaBody(requestBody *openapi3.RequestBodyRef) bool {
	if requestBody.Value.Content == nil {
		return false
	}
	switch requestBody.Value.Content[binding.MIMEJSON].Schema.Value.Type {
	case "object":
		return len(requestBody.Value.Content[binding.MIMEJSON].Schema.Value.Properties) != 0
	case "array":
		return len(requestBody.Value.Content[binding.MIMEJSON].Schema.Value.Items.Value.Properties) != 0
	}

	return false
}

func (swagger *Swagger) getPaths() *openapi3.Paths {
	paths := &openapi3.Paths{Extensions: make(map[string]any)}
	for path, m := range swagger.Routers {
		pathItem := &openapi3.PathItem{}
		for method, r := range m {
			if r.Exclude {
				continue
			}
			model := r.Model
			operation := &openapi3.Operation{
				Tags:        r.Tags,
				OperationID: r.OperationID,
				Summary:     r.Summary,
				Description: r.Description,
				Deprecated:  r.Deprecated,
				Responses:   swagger.getResponses(r.Response, r.ResponseContentType),
				Parameters:  swagger.getParametersByModel(model),
				Security:    swagger.getSecurityRequirements(r.Securities),
			}
			body := reflect.ValueOf(model).FieldByName("Body")
			if body.IsValid() {
				bodyValue := body.Interface()
				requestBody := swagger.getRequestBodyByModel(bodyValue, r.RequestContentType)
				if swagger.hasSchemaBody(requestBody) {
					operation.RequestBody = requestBody
				}
			}

			if method == http.MethodGet {
				pathItem.Get = operation
			} else if method == http.MethodPost {
				pathItem.Post = operation
			} else if method == http.MethodDelete {
				pathItem.Delete = operation
			} else if method == http.MethodPut {
				pathItem.Put = operation
			} else if method == http.MethodPatch {
				pathItem.Patch = operation
			} else if method == http.MethodHead {
				pathItem.Head = operation
			} else if method == http.MethodOptions {
				pathItem.Options = operation
			} else if method == http.MethodConnect {
				pathItem.Connect = operation
			} else if method == http.MethodTrace {
				pathItem.Trace = operation
			}
		}
		if pathItem.Get != nil || pathItem.Post != nil || pathItem.Delete != nil || pathItem.Put != nil || pathItem.Patch != nil || pathItem.Head != nil || pathItem.Options != nil || pathItem.Connect != nil || pathItem.Trace != nil {
			paths.Set(swagger.fixPath(path), pathItem)
		}
	}
	return paths
}
func (swagger *Swagger) BuildOpenAPI() {
	components := openapi3.NewComponents()
	components.SecuritySchemes = openapi3.SecuritySchemes{}
	swagger.OpenAPI = &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:          swagger.Title,
			Description:    swagger.Description,
			TermsOfService: swagger.TermsOfService,
			Contact:        swagger.Contact,
			License:        swagger.License,
			Version:        swagger.Version,
		},
		Servers:    swagger.Servers,
		Components: &components,
	}
	swagger.OpenAPI.Paths = swagger.getPaths()
}

func (swagger *Swagger) MarshalJSON() ([]byte, error) {
	return swagger.OpenAPI.MarshalJSON()
}

func (swagger *Swagger) MarshalYAML() ([]byte, error) {
	b, err := swagger.OpenAPI.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var data any
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return yaml.Marshal(data)
}

func (swagger *Swagger) WithDocsUrl(url string) *Swagger {
	DocsUrl(url)(swagger)
	return swagger
}
func (swagger *Swagger) WithRedocUrl(url string) *Swagger {
	RedocUrl(url)(swagger)
	return swagger
}
func (swagger *Swagger) WithTitle(title string) *Swagger {
	Title(title)(swagger)
	return swagger
}
func (swagger *Swagger) WithDescription(description string) *Swagger {
	Description(description)(swagger)
	return swagger
}
func (swagger *Swagger) WithVersion(version string) *Swagger {
	Version(version)(swagger)
	return swagger
}
func (swagger *Swagger) WithOpenAPIUrl(url string) *Swagger {
	OpenAPIUrl(url)(swagger)
	return swagger
}
func (swagger *Swagger) WithTermsOfService(termsOfService string) *Swagger {
	TermsOfService(termsOfService)(swagger)
	return swagger
}
func (swagger *Swagger) WithContact(contact *openapi3.Contact) *Swagger {
	Contact(contact)(swagger)
	return swagger
}
func (swagger *Swagger) WithLicense(license *openapi3.License) *Swagger {
	License(license)(swagger)
	return swagger
}
func (swagger *Swagger) WithServers(servers []*openapi3.Server) *Swagger {
	Servers(servers)(swagger)
	return swagger
}
func (swagger *Swagger) WithSwaggerOptions(options map[string]any) *Swagger {
	SwaggerOptions(options)(swagger)
	return swagger
}
func (swagger *Swagger) WithRedocOptions(options map[string]any) *Swagger {
	RedocOptions(options)(swagger)
	return swagger
}
