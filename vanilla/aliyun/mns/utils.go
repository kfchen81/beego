package mns

import (
	"io/ioutil"
	"net/http"
	
	"github.com/kfchen81/beego/vanilla/gogap/errors"
)

func send(client MNSClient, decoder MNSDecoder, method Method, headers map[string]string, message interface{}, resource string, v interface{}) (statusCode int, err error) {
	var resp *http.Response
	if resp, err = client.Send(method, headers, message, resource); err != nil {
		return
	}
	
	defer resp.Body.Close()

	if resp != nil {
		statusCode = resp.StatusCode

		if statusCode != http.StatusCreated &&
			statusCode != http.StatusOK &&
			statusCode != http.StatusNoContent {

			// get the response body
			//   the body is set in error when decoding xml failed
			//获取response的内容
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return statusCode, err
			}

			var e2 error
			err, e2 = decoder.DecodeError(bodyBytes, resource)

			if e2 != nil {
				err = ERR_UNMARSHAL_ERROR_RESPONSE_FAILED.New(errors.Params{"err": e2, "resp":string(bodyBytes)})
				return statusCode, err
			}
			return statusCode, err
		}

		if v != nil {
			if e := decoder.Decode(resp.Body, v); e != nil {
				err = ERR_UNMARSHAL_RESPONSE_FAILED.New(errors.Params{"err": e})
				return statusCode, err
			}
		}
	}

	return statusCode, err
}
