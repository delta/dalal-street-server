package utils

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
	"encoding/json"
)

var validContent = `
{
   "EventId": "22",
   "EventSecret": "s3kr3t",

   "LogFileName": "./dalalstreet.log",
   "LogMaxSize": 50,
   "LogLevel": "debug",

   "DbUser": "root",
   "DbPassword": "",
   "DbHost": "localhost",
   "DbName": "dalalstreet"
}
`

func TestInitConfiguration(t *testing.T) {
	err := os.Rename("config.json", "config.json.bkp")

	// It's okay if config.json did not exist before
	// Something else is not
	if err != nil && !os.IsNotExist(err) {
		t.Fatal("Unexpected error occured: ", err)
	} else {
		defer os.Rename("config.json.bkp", "config.json")
	}

	f, err := os.Create("config.json")
	f.WriteString(validContent)
	f.Close()

	InitConfiguration()

	loadedConfig, _ := json.Marshal(Configuration)

	assert.JSONEq(t, string(loadedConfig), validContent, "config.json parsed correctly")
}
