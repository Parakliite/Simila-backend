package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

func (r Runtime) MarshalJSON() ([]byte, error) {
	jsVal := fmt.Sprintf("%d min", r)

	quotedJSONValue := strconv.Quote(jsVal)

	return []byte(quotedJSONValue), nil
}
