package helpers

import (
	"github.com/golang/glog"

	"github.com/microcosm-cc/goconfig"
)

const CONFIG_FILE string = "/etc/microcosm/api.conf"

const SECTION_API string = "api"

var (
	KEY_ENVIRONMENT string = "environment"

	KEY_DATABASE_HOST     string = "database_host"
	KEY_DATABASE_PORT     string = "database_port"
	KEY_DATABASE_DATABASE string = "database_database"
	KEY_DATABASE_USERNAME string = "database_username"
	KEY_DATABASE_PASSWORD string = "database_password"

	KEY_MICROCOSM_DOMAIN string = "microcosm_domain"

	KEY_LISTEN_PORT string = "listen_port"

	KEY_MEMCACHED_HOST string = "memcached_host"
	KEY_MEMCACHED_PORT string = "memcached_port"

	KEY_AWS_ACCESS_KEY_ID     string = "aws_access_key_id"
	KEY_AWS_SECRET_ACCESS_KEY string = "aws_secret_access_key"
	KEY_S3_BUCKET             string = "s3_bucket"

	KEY_MAILGUN_API_URL string = "mailgun_api_url"
	KEY_MAILGUN_API_KEY string = "mailgun_api_key"

	KEY_ERROR_LOG_FILENAME string = "error_log_filename"
	KEY_WARN_LOG_FILENAME  string = "warn_log_filename"
	KEY_DEBUG_LOG_FILENAME string = "debug_log_filename"

	KEY_ELASTICSEARCH_HOST string = "elasticsearch_host"
	KEY_ELASTICSEARCH_PORT string = "elasticsearch_port"

	KEY_PERSONA_VERIFIER_URL string = "persona_verifier_url"
)

var configRequiredStrings = []string{
	KEY_AWS_ACCESS_KEY_ID,
	KEY_AWS_SECRET_ACCESS_KEY,
	KEY_DATABASE_DATABASE,
	KEY_DATABASE_HOST,
	KEY_DATABASE_PASSWORD,
	KEY_DATABASE_USERNAME,
	KEY_DEBUG_LOG_FILENAME,
	KEY_DEBUG_LOG_FILENAME,
	KEY_ELASTICSEARCH_HOST,
	KEY_ELASTICSEARCH_HOST,
	KEY_ENVIRONMENT,
	KEY_ERROR_LOG_FILENAME,
	KEY_MAILGUN_API_KEY,
	KEY_MAILGUN_API_URL,
	KEY_MEMCACHED_HOST,
	KEY_MICROCOSM_DOMAIN,
	KEY_PERSONA_VERIFIER_URL,
	KEY_S3_BUCKET,
	KEY_WARN_LOG_FILENAME,
}

var configRequiredInt64s = []string{
	KEY_DATABASE_PORT,
	KEY_ELASTICSEARCH_PORT,
	KEY_LISTEN_PORT,
	KEY_MEMCACHED_PORT,
}

var CONFIG_STRING = map[string]string{}

var CONFIG_INT64 = map[string]int64{}

var CONFIG_BOOL = map[string]bool{}

func init() {

	c, err := goconfig.ReadConfigFile(CONFIG_FILE)
	if err != nil {
		glog.Fatal(err)
	}

	for _, key := range configRequiredStrings {
		s, err := c.GetString(SECTION_API, key)
		if err != nil {
			glog.Fatal(err)
		}
		CONFIG_STRING[key] = s
	}

	for _, key := range configRequiredInt64s {
		ii, err := c.GetInt64(SECTION_API, key)
		if err != nil {
			glog.Fatal(err)
		}
		CONFIG_INT64[key] = ii
	}
}
