package vanilla

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/kfchen81/beego"
	beeContext "github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/metrics"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RestProxy struct {
	RestResource
}

func (this *RestProxy) Resource() string {
	return "ws.rest_proxy"
}

const (
	// Time allowed to writer data to the client.
	writeWait = 10 * time.Second

	// Time allowed to read data to the client.
	readWait = 60 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = readWait

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 8) / 10
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header["Origin"]
		if len(origin) == 0 {
			return true
		}
		u, err := url.Parse(origin[0])
		if err != nil {
			return false
		}
		return HostDomain(u.Host) == HostDomain(r.Host)
	},
}

type RestRequest struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Params string `json:"params"`
	Rid    string `json:"rid"`
}

func (this *RestProxy) Get() {
	ws, err := upgrader.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil)
	if err != nil {
		metrics.GetRestwsErrorCounter().WithLabelValues("upgrade").Inc()
		beego.HError(this.Ctx.Request, err)
		return
	}

	metrics.GetRestwsGauge().Inc()

	respChan := make(chan WsResponse)
	stopRespCh := make(chan struct{})
	closeCh := make(chan struct{})
	defer func() {
		close(closeCh)
		ws.Close()
		metrics.GetRestwsGauge().Dec()
	}()
	go writer(ws, respChan, stopRespCh, closeCh)
	reader(ws, this.Ctx, respChan, stopRespCh)
}

func reader(ws *websocket.Conn, ctx *beeContext.Context, respChan chan<- WsResponse, stopRespCh chan struct{}) {
	for {
		req := new(RestRequest)
		ws.SetReadDeadline(time.Now().Add(readWait))
		err := ws.ReadJSON(req)
		if err != nil {
			beego.Info("Read Error:", err)
			metrics.GetRestwsErrorCounter().WithLabelValues("reader").Inc()
			break
		}
		go handle(ctx, req, respChan, stopRespCh)
	}
}

func handle(ctx *beeContext.Context, req *RestRequest, respChan chan<- WsResponse, stopRespCh chan struct{}) {
	defer func() {
		err := recover()
		if err != nil {
			beego.Error("Handle Error:", err)
			metrics.GetRestwsErrorCounter().WithLabelValues("handle").Inc()
		}
	}()
	log(req)
	resp := handleRequest(*req, ctx)
	select {
	case <-stopRespCh:
		return
	case respChan <- resp:
	}
}

func writer(ws *websocket.Conn, respChan <-chan WsResponse, stopRespCh chan struct{}, closeCh chan struct{}) {
	defer func() {
		err := recover()
		if err != nil {
			beego.Error("Write Recover:", err)
			metrics.GetRestwsErrorCounter().WithLabelValues("writer_panic").Inc()
		}
	}()
	defer func() {
		metrics.GetRestwsErrorCounter().WithLabelValues("writer_close").Inc()
		close(stopRespCh)
		ws.Close()
	}()
	for {
		select {
		case <- closeCh:
			return
		case resp := <-	respChan:
			content, err := json.Marshal(resp)
			if err != nil {
				beego.Error("Encode websocket data error:", err)
				return
			}
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, content); err != nil {
				beego.Info("Write Error:", err)
				metrics.GetRestwsErrorCounter().WithLabelValues("writer").Inc()
				return
			}
		}
	}
}

func log(req *RestRequest) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if !strings.HasPrefix(req.Path, "/") {
		req.Path = fmt.Sprintf("/%s", req.Path)
	}
	beego.Info(fmt.Sprintf("[%s] Method:%s Path:%s Params:%s Rid:%s",
		now, req.Method, req.Path, req.Params, req.Rid))
}
