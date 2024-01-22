package middleware

import (
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/vanilla/encrypt"
)

var ExtractCorpTokenFilter = func(ctx *context.Context) {
	cookie, err := ctx.Request.Cookie("__cs")
	if err != nil {
		//beego.Error(err.Error())
		return
	}

	encodedToken := cookie.Value
	if encodedToken == "" {
		return
	}

	uuid, openid, err := encrypt.DecodeToken(encodedToken)
	if err != nil {
		beego.Error(err.Error())
		return
	}

	ctx.Input.SetData("__corp_token", fmt.Sprintf("%s____%s", uuid, openid))
}

func init() {

}