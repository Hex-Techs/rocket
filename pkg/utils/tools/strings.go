package tools

import (
	"math/rand"
	"time"
)

// ContainsString returns true if the string is in the slice
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString removes the string from the slice
func RemoveString(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// RandomStr generate a random string of n length
func RandomStr(length int) string {
	if length <= 0 {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	var letters = []byte("abcdefghjkmnpqrstuvwxyz123456789")
	var result []byte = make([]byte, length)
	for ix, _ := range result {
		result[ix] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}
