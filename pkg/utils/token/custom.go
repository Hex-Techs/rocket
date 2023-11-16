package token

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GenerateCustomToken 生成一个url类型token
func GenerateCustomToken(str string, exp int) string {
	if exp <= 0 {
		return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:forever", str)))
	}
	return base64.URLEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%d", str, time.Now().Add(time.Second*time.Duration(exp)).Unix())))
}

// ParseCustomToken validate url token
func ParseCustomToken(token string) (string, error) {
	uDec, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	u := strings.Split(string(uDec), ":")
	var expire int64
	tokenErr := fmt.Errorf("token is invalid: %s", token)
	if len(u) <= 1 {
		return "", tokenErr
	}
	if u[1] == "forever" {
		expire = 0
	} else {
		expire, err = strconv.ParseInt(u[1], 10, 64)
		if err != nil {
			return "", err
		}
		if expire <= time.Now().Unix() {
			return "", tokenErr
		}
	}
	return u[0], nil
}
