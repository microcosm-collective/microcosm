package main

import (
	"context"
	"flag"
	"fmt"
	"runtime"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	"github.com/microcosm-collective/microcosm/cache"
	conf "github.com/microcosm-collective/microcosm/config"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/server"
)

var (
	version string
	date    string
)

func main() {
	// Parse flags and start memory profiling
	// Also used to init glog
	flag.Parse()

	// 100 megabytes max before rolling the config files
	glog.MaxSize = 1024 * 1024 * 100

	if glog.V(2) {
		glog.Info(
			fmt.Sprintf(
				`Initialising profiling of %s to %s`,
				conf.ConfigStrings[conf.PyroscopeApp],
				conf.ConfigStrings[conf.PyroscopeAddress],
			),
		)
	}
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)
	_, err := pyroscope.Start(
		pyroscope.Config{
			ApplicationName:   conf.ConfigStrings[conf.PyroscopeApp],
			ServerAddress:     conf.ConfigStrings[conf.PyroscopeAddress],
			BasicAuthUser:     conf.ConfigStrings[conf.PyroscopeUser],
			BasicAuthPassword: conf.ConfigStrings[conf.PyroscopePassword],

			ProfileTypes: []pyroscope.ProfileType{
				// these profile types are enabled by default:
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,

				// these profile types are optional:
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		},
	)
	if err != nil {
		// We are not returning as this is a profiling error and not an application error
		glog.Error(err.Error())
	}

	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("build_version", version, "build_date", date), func(context.Context) {

		// We read the config file (by importing it) and it's our responsibility to
		// set up the database connection and memcache before we start the server
		if glog.V(2) {
			glog.Info(
				fmt.Sprintf(
					`Initialising DB connection on %s:%d for database %s`,
					conf.ConfigStrings[conf.DatabaseHost],
					conf.ConfigInt64s[conf.DatabasePort],
					conf.ConfigStrings[conf.DatabaseName],
				),
			)
		}
		h.InitDBConnection(h.DBConfig{
			Host:     conf.ConfigStrings[conf.DatabaseHost],
			Port:     conf.ConfigInt64s[conf.DatabasePort],
			Database: conf.ConfigStrings[conf.DatabaseName],
			Username: conf.ConfigStrings[conf.DatabaseUsername],
			Password: conf.ConfigStrings[conf.DatabasePassword],
		})

		if glog.V(2) {
			glog.Info(
				fmt.Sprintf(
					`Initialising cache connection to %s:%d`,
					conf.ConfigStrings[conf.MemcachedHost],
					conf.ConfigInt64s[conf.MemcachedPort],
				),
			)
		}
		cache.InitCache(
			conf.ConfigStrings[conf.MemcachedHost],
			conf.ConfigInt64s[conf.MemcachedPort],
		)

		if glog.V(2) {
			glog.Infof(
				"Starting server on port %d",
				conf.ConfigInt64s[conf.ListenPort],
			)
		}

		server.StartServer(conf.ConfigInt64s[conf.ListenPort])
	})
}
