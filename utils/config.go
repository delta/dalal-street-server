package utils

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Configuration contains all the configuration options
var Configuration = struct {
	// Environment related options

	// Stage is the current execution environment. Can be one of "prod", "dev" or "test"
	Stage string

	// Pragyan API related options

	// EventId is the Id of DalalStreet event
	EventId string
	// EventSecret is the secret string for DalalStreet event for Pragyan API
	EventSecret string

	// Logging related options

	// LogFileName is the name of the log file name
	LogFileName string
	// LogMaxSize is the maximum size(MB) of a log file before it gets rotated
	LogMaxSize int
	// LogLevel determines the log level.
	// Can be one of "debug", "info", "warn", "error"
	LogLevel string

	// Database related options

	// DbUser is the name of the database user
	DbUser string
	// DbPassword is the password of the database user
	DbPassword string
	// DbHost is the host name of the database server
	DbHost string
	// DbName is the name of the database
	DbName string

	// HTTP Server related options

	// HttpPort is the port on which the http server will run
	HttpPort int
}{}

// InitConfiguration reads the config.json file and loads the
// config options into Configuration
func init() {
	configFileName := *flag.String("config", "config.json", "Name of the config file")
	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatalf("Failed to open %s. Cannot proceed", configFileName)
		return
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&Configuration)

	if err != nil {
		log.Fatalf("Failed to load configuration. Cannot proceed. Error: ", err)
	}

	log.Printf("Loaded configuration from %s: %+v\n", configFileName, Configuration)
}
