package vanilla

import (
	"crypto/hmac"
	"errors"
	"github.com/bitly/go-simplejson"
	"strings"
	"time"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const SALT string = "030e2cf548cf9da683e340371d1a74ee"

func EncodeJWT(data Map) string {
	//header := "{'typ':'JWT','alg':'HS256'}"
	headerBase64Code := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9"
	
	now := time.Now()
	data["exp"] = Strftime(&now, "%Y-%m-%d %H:%M")
	
	timedelta := Timedelta{Days:365}
	expData := now.Add(timedelta.Duration())
	data["iat"] = Strftime(&expData, "%Y-%m-%d %H:%M")
	
	payload := ToJsonString(data)
	payloadBase64Code := base64.StdEncoding.EncodeToString([]byte(payload))
	
	message := fmt.Sprintf("%s.%s", headerBase64Code, payloadBase64Code)
	h := hmac.New(sha256.New, []byte(SALT))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	return fmt.Sprintf("%s.%s.%s", headerBase64Code, payloadBase64Code, signature)
}

func DecodeJWT(jwtToken string) (*simplejson.Json, error){
	var js *simplejson.Json

	items := strings.Split(jwtToken, ".")
	if len(items) != 3 {
		return js, errors.New(fmt.Sprintf("无效的jwt token 1 - [%s]", jwtToken))
	}

	headerB64Code, payloadB64Code, expectedSignature := items[0], items[1], items[2]
	message := fmt.Sprintf("%s.%s", headerB64Code, payloadB64Code)

	h := hmac.New(sha256.New, []byte(SALT))
	h.Write([]byte(message))
	actualSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if expectedSignature != actualSignature {
		//jwt token的signature不匹配
		return js, errors.New(fmt.Sprintf("无效的jwt token 2 - [%s]", jwtToken))
	}

	decodeBytes, err := base64.StdEncoding.DecodeString(payloadB64Code)
	if err != nil {
		fmt.Println(err)
	}
	js, err = simplejson.NewJson([]byte(decodeBytes))

	if err != nil {
		return js, errors.New(fmt.Sprintf("无效的jwt token 3 - [%s]", jwtToken))
	}

	return js, nil
}

func ParseUserIdFromJwtToken(jwtToken string) (int, int, error){
	var (
		authUserId int
		userId int
	)

	js, err := DecodeJWT(jwtToken)

	if err != nil{
		return userId, authUserId, err
	}

	return ParseUserIdFromJwtData(js)
}

func ParseUserIdFromJwtData(js *simplejson.Json) (int, int, error){
	var (
		authUserId int
		userId int
	)

	jwtType, err := js.Get("type").Int()
	if err != nil {
		return userId, authUserId, errors.New(fmt.Sprintf("无效的jwt token 4.1 - [%s]", err.Error()))
	}

	switch jwtType {
	case 1:
		userId, err = js.Get("user_id").Int()
		authUserId, err = js.Get("uid").Int()
	case 2:
		userId, err = js.Get("uid").Int()
		authUserId = 0
	case 3:
		userId, err = js.Get("user").Get("uid").Int()
		authUserId, err = js.Get("corp_user").Get("uid").Int()
	default:
		err = errors.New(fmt.Sprintf("invalid jwt type: %d", jwtType))
	}
	if err != nil {
		return userId, authUserId, errors.New(fmt.Sprintf("无效的jwt token 4.2 - [%s]", err.Error()))
	}

	return userId, authUserId, nil
}