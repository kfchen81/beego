package vanilla

import (
	"github.com/bitly/go-simplejson"
	"context"
	"net/http"
)

type IModel interface {
	GetId() int
}

type IIDable interface {
	GetId(idType string) int
}

type IBusinessContextFactory interface {
	NewContext(ctx context.Context, request *http.Request, userId int, jwtToken string, rawData *simplejson.Json) context.Context
}