package tools

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

func GenerateName(prefix, name string) string {
	return fmt.Sprintf("%s-%s", prefix, name)
}

func GenerateNameWithHash(prefix, name string) string {
	n := fmt.Sprintf("%s-%s", prefix, name)
	h := sha1.New()
	h.Write([]byte(n))
	sl := strings.Split(fmt.Sprintf("%x", h.Sum(nil)), "")
	return n + "-" + strings.Join(sl[0:10], "")
}
