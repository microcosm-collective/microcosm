package server

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/nytimes/gziphandler"
	"github.com/robfig/cron"

	conf "github.com/microcosm-cc/microcosm/config"
)

// StartServer owns the http process and cron jobs
func StartServer(port int64) {

	// Set up the cron jobs
	c := cron.New()
	for schedule, job := range jobs {
		c.AddFunc(schedule, job)
	}
	c.Start()

	r := mux.NewRouter()

	// Register all handlers for the root site (e.g. http://microco.sm)
	for url, handler := range rootHandlers {
		r.HandleFunc(url, handler).Host(conf.ConfigStrings[conf.MicrocosmDomain])
	}

	// Register all handlers for sites (e.g. http://{[A-Za-z0-9]+}.microco.sm)
	for url, handler := range siteHandlers {
		r.HandleFunc(url, handler).Host("{subdomain:[a-z0-9]+}." + conf.ConfigStrings[conf.MicrocosmDomain])
	}

	http.Handle("/", gziphandler.GzipHandler(r))

	// Start the HTTP server
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
