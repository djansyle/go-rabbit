package rabbit

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

// RandomID generates a random id from uuidv4
func RandomID() string {
	queue := uuid.NewV4()
	return strings.Replace(queue.String(), "-", "", -1)
}
