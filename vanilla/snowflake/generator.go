// Package snowflake provides a very simple Twitter snowflake generator and parser.
// copy from: github.com/bwmarrin/snowflake

package snowflake

import (
	"context"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/vanilla"
	"math/rand"
	"time"
)

var name2node = make(map[string]*Node)

func GetNode(name string) *Node {
	return name2node[name]
}

func getNodeId(nodeKey string) int64 {
	ctx := context.Background()
	reply, err := vanilla.Redis.Do(ctx, "INCR", nodeKey)
	var nodeId int64 = 1
	if err != nil { // 从redis获取失败，则从随机生成
		beego.Error(err)
		rand.Seed(time.Now().UnixNano())
		nodeId = rand.Int63n(nodeMax - 1)
	} else {
		nodeId = reply.(int64)
		if nodeId >= nodeMax {
			nodeId = nodeId % nodeMax // 超出1023则求模
		}
	}
	beego.Info(fmt.Sprintf("[snowflake] get node id %d", nodeId))
	return nodeId
}

func InitNode(name string) error {
	nodekey := "__id_generator"
	node, err := NewNode(getNodeId(nodekey))
	if err != nil {
		beego.Error(err)
		return err
	} else {
		name2node[name] = node
	}

	return nil
}

func InitNodeForService(name, serviceName string) error {
	nodekey := fmt.Sprintf("__id_generator_%s", serviceName)
	node, err := NewNode(getNodeId(nodekey))
	if err != nil {
		beego.Error(err)
		return err
	} else {
		name2node[name] = node
	}

	return nil
}
