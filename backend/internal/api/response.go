package api

import "github.com/gin-gonic/gin"

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Response{Code: 0, Msg: "ok", Data: data})
}

func Fail(c *gin.Context, status, code int, msg string) {
	c.AbortWithStatusJSON(status, Response{Code: code, Msg: msg})
}
