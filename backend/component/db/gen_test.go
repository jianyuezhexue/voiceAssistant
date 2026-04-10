package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTableInfo(t *testing.T) {
	data, err := GetTableInfo("sms", "test")
	dataStr, _ := json.MarshalIndent(data, "", "  ")
	t.Log(string(dataStr))
	assert.Nil(t, err)
}
