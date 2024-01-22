package vanilla

type Map = map[string]interface{}
type FillOption = map[string]bool

type Response2 struct {
	Code        int32                  `json:"code"`
	Data        map[string]interface{} `json:"data"`
	ErrCode string `json:"errCode"`
	ErrMsg      string                 `json:"errMsg"`
	InnerErrMsg string                 `json:"innerErrMsg"`
}

type Response struct {
	Code        int32                  `json:"code"`
	Data        interface{} `json:"data"`
	ErrCode string `json:"errCode"`
	ErrMsg      string                 `json:"errMsg"`
	InnerErrMsg string                 `json:"innerErrMsg"`
	MachineInfo map[string]interface{} `json:"_pod"`
}

func MakeResponse2(data map[string]interface{}) *Response {
	return &Response{
		200,
		data,
		"",
		"",
		"",
		GetMachineInfo(),
	}
}

func MakeResponse(data interface{}) *Response {
	return &Response{
		200,
		data,
		"",
		"",
		"",
		GetMachineInfo(),
	}
}

func MakeErrorResponse(code int32, errCode string, errMsg string, innerErrMsgs ...string) *Response {
	innerErrMsg := ""
	if len(innerErrMsgs) > 0 {
		innerErrMsg = innerErrMsgs[0]
	}

	return &Response{
		code,
		nil,
		errCode,
		errMsg,
		innerErrMsg,
		GetMachineInfo(),
	}
}
