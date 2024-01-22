package engine

var Type2Engine map[string]iEngine

type iEngine interface {
	Send(data map[string]interface{}, tag string)
}

func registerEngine(engineType string, eg iEngine){
	if Type2Engine == nil{
		Type2Engine = make(map[string]iEngine)
	}
	Type2Engine[engineType] = eg
}