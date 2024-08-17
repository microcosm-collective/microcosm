package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"

	"git.dee.kitchen/buro9/microcosm/cache"
	conf "git.dee.kitchen/buro9/microcosm/config"
	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/server"
)

func main() {
	// Parse flags and start memory profiling
	// Also used to init glog
	flag.Parse()

	// 100 megabytes max before rolling the config files
	glog.MaxSize = 1024 * 1024 * 100

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
}
