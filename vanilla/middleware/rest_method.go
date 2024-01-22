package middleware

import (
	"strings"

	"github.com/kfchen81/beego/context"
)

var ModifyRestMethodFilter = func(ctx *context.Context) {
	if ctx.Input.Query("_method") != "" && ctx.Input.IsPost() {
		ctx.Request.Method = strings.ToUpper(ctx.Input.Query("_method"))
	}
}
