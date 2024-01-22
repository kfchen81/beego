package vanilla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kfchen81/beego"
	"io/ioutil"
	"net/http"
)

var mode string
var platformName string
var envName string

type DingBot struct{
	apiUrl string
	token string
	botName string
}

func (this *DingBot) SetToken(token string) *DingBot{
	this.token = token
	return this
}

func (this *DingBot) Use(name string) *DingBot{
	botConf, err := this.parseConf()
	if err != nil{
		beego.Error(err)
	}else{
		if token, ok := botConf[name]; ok{
			this.token = token.(string)
			this.botName = name
		}
	}

	return this
}

func (this *DingBot) parseConf() (map[string]interface{}, error){
	botStr := beego.AppConfig.String("ding::BOT")
	var botSetting map[string]interface{}
	err := json.Unmarshal([]byte(botStr), &botSetting)
	if err != nil{
		return botSetting, err
	}
	return botSetting, nil
}

/*
	send 发送钉钉消息
	文本消息格式：
	{
		"msgtype": "markdown",
		"markdown": {
			"title": "首屏会话透出的展示内容",
			"text": "# 这是支持markdown的文本 \n## 标题2  \n* 列表1 \n"
		}
	}
 */
func (this *DingBot) send(title, msg string){
	if this.token == "" || mode == "develop"{
		// 开发环境或token不存在则只打印
		beego.Info(title, msg)
		return
	}

	if mode != "deploy" && this.botName == "xiuer"{
		// 秀儿只在生产环境使用
		beego.Info(title, msg)
		return
	}

	apiUrl := fmt.Sprintf("%s?access_token=%s", this.apiUrl, this.token)

	data := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text": fmt.Sprintf("%s \n > %s%s %s", msg, platformName, envName, title),
		},
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil{
		beego.Error(err)
		return
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		beego.Error(err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

// Info 通知
func (this *DingBot) Info(msg string){
	this.send("通知...", msg)
}

// Warn 警告
func (this *DingBot) Warn(msg string){
	this.send("警告-_-", msg)
}

// Error 错误
func (this *DingBot) Error(msg string){
	this.send("错误>_<", msg)
}

// Critical 严重错误
func (this *DingBot) Critical(msg string){
	this.send("严重错误！！！", msg)
}

func NewDingBot() *DingBot{
	bot := new(DingBot)
	bot.apiUrl = "https://oapi.dingtalk.com/robot/send"
	return bot
}

func init(){
	mode = beego.AppConfig.String("ding::DINGBOT_MODE")
	if cn, ok := map[string]string{
		"develop": "开发环境",
		"test": "测试环境",
		"deploy": "生产环境",
	}[mode]; ok{
		envName = cn
	}else{
		envName = "未知环境"
	}
	platformName = beego.AppConfig.String("ding::PLATFORM_NAME")
}