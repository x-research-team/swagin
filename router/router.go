/*
 *   Copyright (c) 2023
 *   All rights reserved.
 */
package router

import (
	"container/list"
	"log"
	"net/http"
	"github.com/goccy/go-reflect"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/x-research-team/swagin/security"
)

type Model any
type ErrorHandlerFunc func(ctx *gin.Context, err error, status int)

type Router struct {
	Handlers            *list.List
	Path                string
	Method              string
	Summary             string
	Description         string
	Deprecated          bool
	RequestContentType  string
	ResponseContentType string
	Tags                []string
	API                 gin.HandlerFunc
	Model               Model
	OperationID         string
	Exclude             bool
	Securities          []security.ISecurity
	Response            Response
}

var validate = validator.New()

func BindModel(req any) gin.HandlerFunc {
	return func(c *gin.Context) {
		m := reflect.New(reflect.TypeOf(req).Elem())
		if m.Kind() == reflect.Ptr {
			m = m.Elem()
		}
		header := m.FieldByName("Header")
		if header.IsValid() {
			headerValue := header.Interface()
			if err := c.ShouldBindWith(&headerValue, Header); err != nil {
				log.Panic(err)
			}
			header.Set(reflect.ValueOf(headerValue))
		}

		query := m.FieldByName("Query")
		if query.IsValid() {
			queryValue := query.Interface()
			if err := c.ShouldBindWith(&queryValue, Query); err != nil {
				log.Panic(err)
			}
			query.Set(reflect.ValueOf(queryValue))
		}
		body := m.FieldByName("Body")
		if body.IsValid() {
			bodyReq := body.Interface()
			bodyValue := body.Interface()
			if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
				switch c.Request.Header.Get("Content-Type") {
				case binding.MIMEMultipartPOSTForm:
					if err := c.ShouldBindWith(&bodyValue, binding.FormMultipart); err != nil {
						log.Panic(err)
					}
				case binding.MIMEJSON:
					if err := c.ShouldBindWith(&bodyValue, binding.JSON); err != nil {
						log.Panic(err)
					}
				case binding.MIMEXML:
					if err := c.ShouldBindWith(&bodyValue, binding.XML); err != nil {
						log.Panic(err)
					}
				case binding.MIMEPOSTForm:
					if err := c.ShouldBindWith(&bodyValue, binding.Form); err != nil {
						log.Panic(err)
					}
				case binding.MIMEYAML:
					if err := c.ShouldBindWith(&bodyValue, binding.YAML); err != nil {
						log.Panic(err)
					}
				case binding.MIMEPROTOBUF:
					if err := c.ShouldBindWith(&bodyValue, binding.ProtoBuf); err != nil {
						log.Panic(err)
					}
				case binding.MIMEMSGPACK:
					if err := c.ShouldBindWith(&bodyValue, binding.MsgPack); err != nil {
						log.Panic(err)
					}
				}
			}
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				TagName: "json",
				Result:  &bodyReq,
			})
			if err != nil {
				log.Panic(err)
			}
			if err := decoder.Decode(bodyValue); err != nil {
				log.Panic(err)
			}
			body.Set(reflect.ValueOf(bodyReq))
		}

		uri := m.FieldByName("URI")
		if uri.IsValid() {
			uriValue := uri.Interface()
			uriMap := make(map[string]string)
			if err := c.ShouldBindUri(&uriMap); err != nil {
				log.Panic(err)
			}
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				TagName: "uri",
				Result:  &uriValue,
			})
			if err != nil {
				log.Panic(err)
			}
			if err := decoder.Decode(uriMap); err != nil {
				log.Panic(err)
			}
			uri.Set(reflect.ValueOf(uriValue))
		}

		model := m.Interface()

		if err := validate.Struct(model); err != nil {
			log.Panic(err)

		}
		if err := copier.Copy(req, model); err != nil {
			log.Panic(err)

		}
		c.Next()
	}
}

func (router *Router) GetHandlers() []gin.HandlerFunc {
	var handlers []gin.HandlerFunc
	for _, s := range router.Securities {
		handlers = append(handlers, s.Authorize)
	}
	for h := router.Handlers.Front(); h != nil; h = h.Next() {
		if f, ok := h.Value.(gin.HandlerFunc); ok {
			handlers = append(handlers, f)
		}
	}
	handlers = append(handlers, router.API)
	return handlers
}

func NewX(f gin.HandlerFunc, options ...Option) *Router {
	r := &Router{
		Handlers: list.New(),
		Response: make(Response),
		API: func(ctx *gin.Context) {
			f(ctx)
		},
	}
	for _, option := range options {
		option(r)
	}
	return r
}
func New[T Model, F func(c *gin.Context, req T)](f F, options ...Option) *Router {
	var model T
	h := BindModel(&model)
	r := &Router{
		Handlers: list.New(),
		Response: make(Response),
		API: func(ctx *gin.Context) {
			f(ctx, model)
		},
		Model: model,
	}
	for _, option := range options {
		option(r)
	}

	r.Handlers.PushBack(h)
	return r
}
func (router *Router) WithSecurity(securities ...security.ISecurity) *Router {
	Security(securities...)(router)
	return router
}
func (router *Router) WithResponses(response Response) *Router {
	Responses(response)(router)
	return router
}
func (router *Router) WithHandlers(handlers ...gin.HandlerFunc) *Router {
	Handlers(handlers...)(router)
	return router
}
func (router *Router) WithTags(tags ...string) *Router {
	Tags(tags...)(router)
	return router
}
func (router *Router) WithSummary(summary string) *Router {
	Summary(summary)(router)
	return router
}
func (router *Router) WithDescription(description string) *Router {
	Description(description)(router)
	return router
}
func (router *Router) WithDeprecated() *Router {
	Deprecated()(router)
	return router
}
func (router *Router) WithOperationID(ID string) *Router {
	OperationID(ID)(router)
	return router
}
func (router *Router) WithExclude() *Router {
	Exclude()(router)
	return router
}
func (router *Router) WithContentType(contentType string, contentTypeType ContentTypeType) *Router {
	ContentType(contentType, contentTypeType)(router)
	return router
}
