package middleware

import (
	"github.com/kfchen81/beego/vanilla"
)

var gBContextFactory vanilla.IBusinessContextFactory

func SetBusinessContextFactory(factory vanilla.IBusinessContextFactory) {
	gBContextFactory = factory
}