package utils

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Config contains all the configuration options
type Config struct {
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

	// GRPC Server related options

	// GrpcAddress is the address to which the gRPC server will bind
	GrpcAddress string
	// GrpcCert is the location of the TLS certificate to be used by the server
	GrpcCert string
	// GrpcKey is the location of the TLS private key to be used by the server
	GrpcKey string
	//CacheSize is the size of the LRU Cache for sessions(As of Now)
	CacheSize int
}

// Struct to load configurations of all possible modes i.e dev, docker, prod, test
// Only one of them will be selected based on the environment variable DALAL_ENV
var AllConfigurations = struct {

	// Configuration for environment : dev
	Dev Config

	// Configuration for environment : docker
	Docker Config

	// Configuration for environment : prod
	Prod Config

	// Configuration for environment : test
	Test Config
}{}

// To store the selected configuration based on the env variable DALAL_ENV
var Configuration Config

// InitConfiguration reads the config.json file and loads the
// config options into Configuration
func init() {
	stage, exists := os.LookupEnv("DALAL_ENV")

	if !exists {
		os.Stderr.WriteString("Set environment variable DALAL_ENV to one of : Dev, Docker, Prod, Test. Taking Dev as default.")
		stage = "Dev"
	}

	configFileName := flag.String("config", "config.json", "Name of the config file")
	flag.Parse()
	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalf("Failed to open %s. Cannot proceed", *configFileName)
		return
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&AllConfigurations)

	if err != nil {
		log.Fatalf("Failed to load configuration. Cannot proceed. Error: ", err)
	}

	switch stage {
	case "Dev":
		Configuration = AllConfigurations.Dev
	case "Docker":
		Configuration = AllConfigurations.Docker
	case "Prod":
		Configuration = AllConfigurations.Prod
	case "Test":
		Configuration = AllConfigurations.Test
	default:
		// Take Dev as default
		Configuration = AllConfigurations.Dev
	}

	log.Printf("Loaded configuration from %s: %+v\n", configFileName, Configuration)
}
