package main

import (
	"context"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/x-research-team/swagin"
)

func main() {
	app := swagin.New(New(), swagin.Server(&http.Server{
		Addr: ":8081",
		BaseContext: func(l net.Listener) context.Context {
			return context.Background()
		},
	})).WithErrorHandler(func(ctx *gin.Context, err error, status int) {
		ctx.AbortWithStatusJSON(status, gin.H{
			"error": err.Error(),
		})
	})
	app.POST("/body", body)

	if err := app.StartGraceful(":8081"); err != nil {
		panic(err)
	}
}
