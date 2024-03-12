package helpers

import (
	"github.com/golang/glog"

	"github.com/robfig/config"
)

// ConfigFilePath is the path to the config file
const ConfigFilePath string = "/etc/microcosm/api.conf"

// APISection is the [api] section of the config file
const APISection string = "api"

// Config file keys
const (
	Environment = "environment"

	DatabaseHost     = "database_host"
	DatabasePort     = "database_port"
	DatabaseName     = "database_database"
	DatabaseUsername = "database_username"
	DatabasePassword = "database_password"

	MicrocosmDomain = "microcosm_domain"

	ListenPort = "listen_port"

	MemcachedHost = "memcached_host"
	MemcachedPort = "memcached_port"

	S3Endpoint        = "s3_endpoint"
	S3BucketName      = "s3_bucket_name"
	S3AccessKeyID     = "s3_access_key_id"
	S3SecretAccessKey = "s3_secret_access_key"

	SendGridAPIKey = "sendgrid_api_key"

	ErrorLogFilename = "error_log_filename"
	WarnLogFilename  = "warn_log_filename"
	DebugLogFilename = "debug_log_filename"
)

var configRequiredStrings = []string{
	DatabaseHost,
	DatabaseName,
	DatabasePassword,
	DatabaseUsername,
	DebugLogFilename,
	Environment,
	ErrorLogFilename,
	MemcachedHost,
	MicrocosmDomain,
	S3AccessKeyID,
	S3BucketName,
	S3Endpoint,
	S3SecretAccessKey,
	SendGridAPIKey,
	WarnLogFilename,
}

var configRequiredInt64s = []string{
	DatabasePort,
	ListenPort,
	MemcachedPort,
}

// ConfigStrings contains the string values for the given config keys
var ConfigStrings = map[string]string{}

// ConfigInt64s contains the int64 values for the given config keys
var ConfigInt64s = map[string]int64{}

// ConfigBool contains the bool values for the given config keys
var ConfigBool = map[string]bool{}

func init() {
	c, err := config.ReadDefault(ConfigFilePath)
	if err != nil {
		glog.Fatal(err)
	}

	for _, key := range configRequiredStrings {
		s, err := c.String(APISection, key)
		if err != nil {
			glog.Fatal(err)
		}
		ConfigStrings[key] = s
	}

	for _, key := range configRequiredInt64s {
		ii, err := c.Int(APISection, key)
		if err != nil {
			glog.Fatal(err)
		}
		ConfigInt64s[key] = int64(ii)
	}
}
