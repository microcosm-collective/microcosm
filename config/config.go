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
	S3UseSsl          = "s3_use_ssl"
	S3BucketName      = "s3_bucket_name"
	S3AccessKeyID     = "s3_access_key_id"
	S3SecretAccessKey = "s3_secret_access_key"

	SendGridAPIKey = "sendgrid_api_key"

	ErrorLogFilename = "error_log_filename"
	WarnLogFilename  = "warn_log_filename"
	DebugLogFilename = "debug_log_filename"

	PyroscopeApp      = "pyroscope_app"
	PyroscopeAddress  = "pyroscope_address"
	PyroscopeUser     = "pyroscope_user"
	PyroscopePassword = "pyroscope_password"
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
	PyroscopeApp,
	PyroscopeAddress,
	PyroscopeUser,
	PyroscopePassword,
}

var configRequiredInt64s = []string{
	DatabasePort,
	ListenPort,
	MemcachedPort,
}

var configRequiredBools = []string{}

// ConfigStrings contains the string values for the given config keys
var ConfigStrings = map[string]string{}

// ConfigInt64s contains the int64 values for the given config keys
var ConfigInt64s = map[string]int64{}

// ConfigBools contains the bool values for the given config keys
var ConfigBools = map[string]bool{}

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

	for _, key := range configRequiredBools {
		bb, err := c.Bool(APISection, key)
		if err != nil {
			glog.Fatal(err)
		}
		ConfigBools[key] = bool(bb)
	}

	ConfigBools[S3UseSsl] = true
	if bb, err := c.Bool(APISection, S3UseSsl); err == nil {
		ConfigBools[S3UseSsl] = bool(bb)
	} else {
		glog.Warningf("%s missing or invalid, defaulting to true: %v", S3UseSsl, err)
	}
}
