package vanilla

import (
	"strconv"

	"github.com/kfchen81/beego/context"
)

//PageInfo 指示当前查询的数据的page信息
type PageInfo struct {
	Page         int
	FromId       int
	CountPerPage int
	Mode         string
	Direction    string
}

//type MobilePageInfo struct {
//	FromId int
//	CountPerPage int
//}

func (self *PageInfo) IsApiServerMode() bool {
	return self.Mode == "apiserver"
}

func (self *PageInfo) Desc() *PageInfo {
	self.Direction = "desc"
	if self.FromId == 0 {
		self.FromId = 99999999999
	}
	return self
}

func (self *PageInfo) Asc() *PageInfo {
	self.Direction = "asc"
	return self
}

func getInt(ctx *context.Context, key string, def ...int) (int, error) {
	strv := ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.Atoi(strv)
}

//ExtractPageInfoFromRequest 从Request中抽取page信息
func ExtractPageInfoFromRequest(ctx *context.Context) *PageInfo {
	fromParam := ctx.Input.Query("_p_from")
	if fromParam == "" {
		fromParam = ctx.Input.Param("_p_from")
	}
	if fromParam != "" {
		fromId, _ := getInt(ctx, "_p_from")
		countPerPage, _ := getInt(ctx, "_p_count", 20)
		return &PageInfo{
			Page:         -1,
			FromId:       fromId,
			CountPerPage: countPerPage,
			Mode:         "apiserver",
			Direction:    "asc",
		}
	} else {
		page, _ := getInt(ctx, "page", 1)
		countPerPage, _ := getInt(ctx, "count_per_page", 20)
		return &PageInfo{
			Page:         page,
			FromId:       0,
			CountPerPage: countPerPage,
			Mode:         "backend",
			Direction:    "asc",
		}
	}
}
