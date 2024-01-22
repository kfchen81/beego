package vanilla

import (
	"context"
	
	//	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/orm"
)

func GetOrmFromContext(ctx context.Context) orm.Ormer {
	o := ctx.Value("orm")
	if o == nil {
		return nil
	}
	return o.(orm.Ormer)
}