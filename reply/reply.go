package reply

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const (
	CodeOK   = 0
	CodeFail = -1
)

func Reply(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, &Message{
		Code: CodeOK,
		Data: data,
	})
}

func ReplyErr(ctx *gin.Context, code int, hints ...string) {
	msg := &Message{
		Code:    code,
		Message: strings.Join(hints, ", "),
		Data:    nil,
	}
	ctx.JSON(http.StatusOK, msg)
}

func ReplyInternalErr(ctx *gin.Context, hints ...string) {
	msg := &Message{
		Code:    CodeFail,
		Message: strings.Join(hints, ", "),
	}
	ctx.JSON(http.StatusOK, msg)
}

type Message struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
