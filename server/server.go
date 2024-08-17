package server

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/robfig/cron"

	conf "git.dee.kitchen/buro9/microcosm/config"
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

	// Register all handlers for the root site (e.g. http://microcosm.app)
	for url, handler := range rootHandlers {
		r.HandleFunc(url, handler).Host(conf.ConfigStrings[conf.MicrocosmDomain])
	}

	// Register all handlers for sites (e.g. http://{[A-Za-z0-9]+}.microcosm.app)
	for url, handler := range siteHandlers {
		r.HandleFunc(url, handler).Host("{subdomain:[a-z0-9]+}." + conf.ConfigStrings[conf.MicrocosmDomain])
	}

	http.Handle("/", r)

	// Start the HTTP server
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
