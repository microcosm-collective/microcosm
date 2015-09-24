package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	// Expose profiling info at /debug/pprof/
	_ "net/http/pprof"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/cache"
	conf "github.com/microcosm-cc/microcosm/config"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/server"
)

var memprof = flag.String("memprof", "", "write memory profile to file")

func main() {

	// Go as fast as we can
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Parse flags and start memory profiling
	// Usage: -memprof=microcosm.mprof
	// Also used to init glog
	flag.Parse()

	// 100 megabytes max before rolling the config files
	glog.MaxSize = 1024 * 1024 * 100

	if *memprof != "" {
		// Reference time is used for formatting.
		// See http://golang.org/pkg/time for details.
		fname := *memprof + "-" + time.Now().Format("2006-01-02_15-04-05-MST")
		f, err := os.Create(fname)
		if err != nil {
			glog.Fatal(err)
		}

		// Catch SIGINT and write heap profile
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		go func() {
			for sig := range c {
				glog.Warningf("Caught %v, stopping profiler and exiting..", sig)
				// Heap profiler is run on GC, so make sure it GCs before exiting.
				runtime.GC()
				pprof.WriteHeapProfile(f)
				f.Close()
				glog.Flush()
				os.Exit(1)
			}
		}()
	} else {
		// Catch closing signal and flush logs
		sigc := make(chan os.Signal, 1)
		signal.Notify(
			sigc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)
		go func() {
			<-sigc
			glog.Flush()
			os.Exit(1)
		}()
	}

	// We read the config file (by importing it) and it's our responsibility to
	// set up the database connection and memcache before we start the server
	if glog.V(2) {
		glog.Info("Initialising DB connection")
	}
	h.InitDBConnection(h.DBConfig{
		Host:     conf.ConfigStrings[conf.DatabaseHost],
		Port:     conf.ConfigInt64s[conf.DatabasePort],
		Database: conf.ConfigStrings[conf.DatabaseName],
		Username: conf.ConfigStrings[conf.DatabaseUsername],
		Password: conf.ConfigStrings[conf.DatabasePassword],
	})

	if glog.V(2) {
		glog.Info("Initialising cache connection")
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
