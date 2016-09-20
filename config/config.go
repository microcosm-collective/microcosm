package helpers

import (
	"github.com/golang/glog"

	"github.com/microcosm-cc/goconfig"
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

	AWSAccessKeyID     = "aws_access_key_id"
	AWSSecretAccessKey = "aws_secret_access_key"
	AWSS3BucketName    = "s3_bucket"

	MailgunAPIURL = "mailgun_api_url"
	MailgunAPIKey = "mailgun_api_key"

	SendGridAPIKey = "sendgrid_api_key"

	ErrorLogFilename = "error_log_filename"
	WarnLogFilename  = "warn_log_filename"
	DebugLogFilename = "debug_log_filename"

	ElasticSearchHost = "elasticsearch_host"
	ElasticSearchPort = "elasticsearch_port"

	PersonaVerifierURL = "persona_verifier_url"
)

var configRequiredStrings = []string{
	AWSAccessKeyID,
	AWSS3BucketName,
	AWSSecretAccessKey,
	DatabaseHost,
	DatabaseName,
	DatabasePassword,
	DatabaseUsername,
	DebugLogFilename,
	ElasticSearchHost,
	Environment,
	ErrorLogFilename,
	MailgunAPIKey,
	MailgunAPIURL,
	SendGridAPIKey,
	MemcachedHost,
	MicrocosmDomain,
	PersonaVerifierURL,
	WarnLogFilename,
}

var configRequiredInt64s = []string{
	DatabasePort,
	ElasticSearchPort,
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
	c, err := goconfig.ReadConfigFile(ConfigFilePath)
	if err != nil {
		glog.Fatal(err)
	}

	for _, key := range configRequiredStrings {
		s, err := c.GetString(APISection, key)
		if err != nil {
			glog.Fatal(err)
		}
		ConfigStrings[key] = s
	}

	for _, key := range configRequiredInt64s {
		ii, err := c.GetInt64(APISection, key)
		if err != nil {
			glog.Fatal(err)
		}
		ConfigInt64s[key] = ii
	}
}
