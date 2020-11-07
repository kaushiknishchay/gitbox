package models

import (
	"bytes"
	"encoding/json"
)

const (
	MetaInfoType_Create string = "CREATE"
	MetaInfoType_Update string = "UPDATE"
	MetaInfoType_Delete string = "DELETE"
)

// MetadataInfo repo update format
type MetadataInfo struct {
	Type   string `json:"type"`
	OldSha string `json:"oldSha"`
	NewSha string `json:"newSha"`
	Ref    string `json:"ref"`
}

// Bytes return the struct as bytes array
func (metadata *MetadataInfo) Bytes() []byte {
	byteBuffer := new(bytes.Buffer)
	_ = json.NewEncoder(byteBuffer).Encode(metadata)

	return byteBuffer.Bytes()
}
