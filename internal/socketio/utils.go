package socketio

import (
	"fmt"
	"github.com/zishang520/socket.io/v2/socket"
)

// Utility function to check arguments length and perform type assertion
func checkArgsAndType[T any](args []any, index int) (T, error) {
	if len(args) <= index {
		return *new(T), fmt.Errorf("missing argument at index %d", index)
	}
	value, ok := args[index].(T)
	if !ok {
		return *new(T), fmt.Errorf("argument at index %d has incorrect type", index)
	}
	return value, nil
}

// Utility function to emit error messages
func emitError(client *socket.Socket, message string, err error) {
	if err != nil {
		client.Emit("error", fmt.Sprintf("%s: %v", message, err))
	} else {
		client.Emit("error", message)
	}
}
