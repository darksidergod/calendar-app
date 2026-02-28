package utils

import (
	"encoding/json"
	"fmt"
)

func PrettyPrint(data interface{}) {
	// Convert data to pretty-printed JSON.
	if out, err := json.MarshalIndent(data, "", "  "); err == nil {
		fmt.Println(string(out))
	} else {
		fmt.Println(err)
	}
}
