package utils

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"testing"
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

	// GRPC Server related options

	// ServerPort is the address to which the gRPC server will bind
	ServerPort string
	// TLSCert is the location of the TLS certificate to be used by the server
	TLSCert string
	// TLSKey is the location of the TLS private key to be used by the server
	TLSKey string
	//CacheSize is the size of the LRU Cache for sessions(As of Now)
	CacheSize int
	//BotSecret is a string used for validation of bots
	BotSecret string

	//SMS API Related options
	PlivoAuthId    string
	PlivoAuthToken string

	//Sendgrid API Key
	SendgridKey string

	// Authentication related options

	// Maximum number of OTP verification requests the user can make without getting blocked
	MaxOTPRequestCount int
	// Expiry time for a generated OTP in minutes
	OTPExpiryTime int
	// Maximum number of times the user can get blocked for a day in the game. After he hits maximum he will be blocked permanently
	MaxBlockCount int
	// Reward for user when someone registers with their referral code
	ReferralCashReward uint64
}

// Struct to load configurations of all possible modes i.e dev, docker, prod, test
// Only one of them will be selected based on the environment variable DALAL_ENV
var allConfigurations = struct {

	// Configuration for environment : dev
	Dev Config

	// Configuration for environment : docker
	Docker Config

	// Configuration for environment : prod
	Prod Config

	// Configuration for environment : test
	Test Config
}{}

// setting config defaults for test, because when running tests
// config.json won't get loaded correctly unless specified by flags
// that gets painful when running individual tests
var config = &Config{
	Stage:              "test",
	EventId:            "1",
	EventSecret:        "be3653b77836f84ab0c1ba3f18abf36e878c5e84",
	LogFileName:        "stdout",
	LogMaxSize:         50,
	LogLevel:           "debug",
	DbUser:             "root",
	DbPassword:         "",
	DbHost:             "",
	DbName:             "dalalstreet_test",
	ServerPort:         ":8000",
	TLSCert:            "./tls_keys/test/server.crt",
	TLSKey:             "./tls_keys/test/server.key",
	CacheSize:          1000,
	BotSecret:          "hellobots",
	PlivoAuthId:        "",
	PlivoAuthToken:     "",
	SendgridKey:        "",
	MaxOTPRequestCount: 100,
	OTPExpiryTime:      5,
	MaxBlockCount:      3,
	ReferralCashReward: 2000,
}

var configFileName *string

// init reads the config.json file and loads the
// config options into config
func init() {
	testing.Init() // as per go 1.13+
	configFileName = flag.String("config", "config.json", "Name of the config file")
	flag.Parse()

	stage, exists := os.LookupEnv("DALAL_ENV")

	if !exists {
		if flag.Lookup("test.v") != nil {
			stage = "Test"
		} else {
			os.Stderr.WriteString("Set environment variable DALAL_ENV to one of : Dev, Docker, Prod, Test. Taking Dev as default.")
			stage = "Dev"
		}
	}

	configFile, err := os.Open(*configFileName)
	if err != nil {
		if stage == "Test" {
			return // config is already set to default value for test. nothing to do.
		}
		log.Fatalf("Failed to open %s. Cannot proceed", *configFileName)
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&allConfigurations)

	if err != nil {
		log.Fatalf("Failed to load configuration. Cannot proceed. Error: %+v", err)
	}

	switch stage {
	case "Dev":
		config = &allConfigurations.Dev
	case "Docker":
		config = &allConfigurations.Docker
	case "Prod":
		config = &allConfigurations.Prod
	case "Test":
		config = &allConfigurations.Test
	default:
		// Take Dev as default
		config = &allConfigurations.Dev
	}

	log.Printf("Loaded configuration from %s: %+v\n", *configFileName, config)
}

// GetConfiguration returns the configuration loaded from config.json
func GetConfiguration() *Config {
	return config
}

// Init intializes the utils package. The config is accepted as a parameter for helping with testing.
func Init(config *Config) {
	initDbHelper(config)
	initLogger(config)
}
