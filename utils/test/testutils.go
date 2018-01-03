package testutils

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func AssertEqual(t *testing.T, o1, o2 interface{}) bool {
	json1, _ := json.Marshal(o1)
	json2, _ := json.Marshal(o2)

	return assert.JSONEq(t, string(json1), string(json2))
}

func Sleep(seconds time.Duration) {
	time.Sleep(seconds * time.Millisecond)
}
