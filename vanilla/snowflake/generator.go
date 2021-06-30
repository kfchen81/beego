// Package snowflake provides a very simple Twitter snowflake generator and parser.
// copy from: github.com/bwmarrin/snowflake

package snowflake

import (
	"context"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/vanilla"
)

var name2node = make(map[string]*Node)

func GetNode(name string) *Node {
	return name2node[name]
}

func getNodeId() int64 {
	ctx := context.Background()
	reply, err := vanilla.Redis.Do(ctx,"INCR", "__id_generator")
	if err != nil {
		beego.Error(err)
		panic("[snowflake] can not get node id from redis")
	} else {
		nodeId := reply.(int64)
		if nodeId > 1000{  // nodeId最大为1042，为避免超过此值，在1000之后重置为0
			_, err = vanilla.Redis.Do(ctx,"SET", "__id_generator", 0)
			if err != nil {
				beego.Error(err)
			}
		}
		beego.Info(fmt.Sprintf("[snowflake] get node id %d", nodeId))
		return nodeId
	}
}

func InitNode(name string) error {
	node, err := NewNode(getNodeId())
	if err != nil {
		beego.Error(err)
		return err
	} else {
		name2node[name] = node
	}
	
	return nil
}