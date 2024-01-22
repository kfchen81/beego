package vanilla

import (
	"fmt"
	"strings"
	"encoding/json"
)

func ConvertToBeegoOrmFilter(filters Map) Map {
	params := make(map[string]interface{})
	var key string
	var match string
	
	for k, v := range filters {
		if !(strings.HasPrefix(k, "__f")) {
			params[k] = v
		} else {
			keyString := strings.Split(k, "-")
			if len(keyString) == 3 {
				switch keyString[2] {
				case "equal":
					key = fmt.Sprintf("%s", keyString[1])
					params[key] = v
					continue
				case "contain":
					match = "contains"
				case "gt", "gte", "lt", "lte", "in":
					match = keyString[2]
				case "range":
					val := v.([]interface{})
					if len(val) > 0{
						switch val[0].(type) {
						case json.Number:
							start, _ := val[0].(json.Number).Int64()
							stop, _ := val[1].(json.Number).Int64()

							values := make([]interface{}, 0)
							for i := start; i < stop; i++ {
								values = append(values, i)
							}
							v = values
						case string:
							switch keyString[1]{
							case "created_at", "updated_at", "finished_at":
								l := keyString[1] + "__gte"
								r := keyString[1] + "__lte"
								params[l] = val[0].(string)
								params[r] = val[1].(string)
								continue
							}
						}
					}
					match = "in"
				default:
					match = keyString[2]
				}
				key = fmt.Sprintf("%s__%s", keyString[1], match)
			} else {
				key = fmt.Sprintf("%s", keyString[1])
			}
			params[key] = v
		}
	}
	return params
}
