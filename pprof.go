// 使用方法
// 在main.go中调用 beego.EnablePProf()
// 确保conf中SKIP_JWT_CHECK_URLS包含/debug/pprof/
// 在本地 go tool pprof -http=:8000 http://xx.xxx.com/{service_name}/debug/pprof/heap

package beego

import (
	"net/http"
	"net/http/pprof"
)

type pHandler struct {
	fn func(http.ResponseWriter, *http.Request)
}

func (ph pHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph.fn(w, r)
}

func wrap(f http.HandlerFunc) http.Handler {
	h := pHandler{}
	h.fn = f
	return h
}

func EnablePProf() {
	//BeeApp.Handlers.Handler("/debug/pprof/", wrap(pprof.Index))
	//BeeApp.Handlers.Handler("/debug/pprof/cmdline", wrap(pprof.Cmdline))
	BeeApp.Handlers.Handler("/debug/pprof/trace", wrap(pprof.Trace))
	BeeApp.Handlers.Handler("/debug/pprof/symbol", wrap(pprof.Symbol))
	BeeApp.Handlers.Handler("/debug/pprof/profile", wrap(pprof.Profile))
	BeeApp.Handlers.Handler("/debug/pprof/heap", pprof.Handler("heap"))
	BeeApp.Handlers.Handler("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	BeeApp.Handlers.Handler("/debug/pprof/block", pprof.Handler("block"))
	BeeApp.Handlers.Handler("/debug/pprof/allocs", pprof.Handler("allocs"))
	BeeApp.Handlers.Handler("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	BeeApp.Handlers.Handler("/debug/pprof/mutex", pprof.Handler("mutex"))
}
