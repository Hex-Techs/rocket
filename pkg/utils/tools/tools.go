package tools

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

// GenerateUUID generates a UUID based on the given fileds of the combined string
func GenerateUUID(resourceKind, cluster, namespace, resourceName string) string {
	// Concatenate the parameters with a colon
	combined := strings.Join([]string{resourceKind, cluster, namespace, resourceName}, ":")

	// Generate a UUID based on the MD5 hash of the combined string
	uuid := uuid.NewMD5(uuid.Nil, []byte(combined))

	// Return the UUID string
	return uuid.String()
}

// GenerateVersionWithTime generates a version string based on the current Unix timestamp
func GenerateVersionWithTime() string {
	// Get the current Unix timestamp
	timestamp := time.Now().Unix()

	// Convert the timestamp to a string
	version := fmt.Sprintf("%d", timestamp)

	// Return the version string
	return version
}
