// Package design embeds the same tokens used by the editable OpenPencil file.
package design

import (
	_ "embed"
	"encoding/json"
)

//go:embed tokens.json
var source []byte

var Tokens = func() struct {
	Colors  map[string]string  `json:"colors"`
	Scalars map[string]float32 `json:"scalars"`
} {
	var t struct {
		Colors  map[string]string  `json:"colors"`
		Scalars map[string]float32 `json:"scalars"`
	}
	if err := json.Unmarshal(source, &t); err != nil {
		panic(err)
	}
	return t
}()
