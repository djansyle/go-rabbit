package rabbit

import (
	"github.com/satori/go.uuid"
	"strings"
)

// RandomID generates a random id from uuidv4
func RandomID() string {
	queue := uuid.NewV4()
	return strings.Replace(queue.String(), "-", "", -1)
}
