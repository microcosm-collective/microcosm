package server

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/robfig/cron"
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

	// Register all handlers for the root site (http://microco.sm)
	for url, handler := range rootHandlers {
		r.HandleFunc(url, handler).Host("microco.sm")
	}

	// Register all handlers for sites (http://{[A-Za-z0-9]+}.microco.sm)
	for url, handler := range siteHandlers {
		r.HandleFunc(url, handler).Host("{subdomain:[a-z0-9]+}.microco.sm")
	}

	http.Handle("/", r)

	// Start the HTTP server
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
