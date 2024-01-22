package encrypt

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/kfchen81/beego"
	"net/url"
	"strings"
)

const _MAGIC_CODE = "<->";

func hexCharCodeToStr(hexCode string) string {
	if hexCode[0] == '0' && (hexCode[1] == 'x' || hexCode[1] == 'X') {
		hexCode = hexCode[2:]
	}
	
	codeLen := len(hexCode)
	if codeLen % 2 != 0 {
		return ""
	}
	
	buf := make([]string, 0)
	for i := 0; i < codeLen; i += 2 {
		curCharCode := "%" + hexCode[i:i+2]
		buf = append(buf, curCharCode)
	}
	
	result, err := url.QueryUnescape(strings.Join(buf, ""))
	if err != nil {
		beego.Error(err)
		return ""
	} else {
		return result
	}
}

func DecodeToken(token string) (string, string, error) {
	count := len(token)
	if count == 0 || count % 2 != 0 {
		return "", "", errors.New("invalid token")
	}
	
	decodedHexBytes := make([]byte, 0)
	for i, char := range token {
		if i % 2 == 0 {
			decodedHexBytes = append(decodedHexBytes, byte(char))
		}
	}
	hexCode := string(decodedHexBytes)
	originStr := hexCharCodeToStr(strings.Trim(hexCode, " "))
	items := strings.Split(originStr, "_<->_")
	return items[0], items[1], nil
}

func strToHexCharCode(str string) string {
	if str == "" {
		return ""
	}
	
	return hex.EncodeToString([]byte(str))
}

func EncodeToken(key1 string, key2 string) string {
	str := fmt.Sprintf("%s_%s_%s", key1, _MAGIC_CODE, key2)
	hexStr := strToHexCharCode(str)
	
	hexStrLen := len(hexStr)
	buf := make([]byte, 0)
	for i := 0; i < hexStrLen; i += 1 {
		buf = append(buf, hexStr[i])
		buf = append(buf, '0')
	}
	
	return strings.ToUpper(string(buf))
}