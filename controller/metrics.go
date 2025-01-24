package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/models"
)

// MetricsController is a web controller
type MetricsController struct{}

// MetricsHandler is a web handler
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/metrics"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := MetricsController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "PUT"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Update handles PUT
func (ctl *MetricsController) Update(c *models.Context) {

	// Hard coded to only work for founders.
	if c.Auth.UserID != 1 && c.Auth.UserID != 2 {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Only founders can manually update metrics: %d", c.Auth.UserID),
			http.StatusForbidden,
		)
		return
	}

	err := models.UpdateMetrics()

	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error updating metrics: %+v", err),
			http.StatusInternalServerError,
		)
		return
	}

	c.RespondWithOK()
}

// Read handles GET
func (ctl *MetricsController) Read(c *models.Context) {

	// Hard coded to only work for founders.
	if c.Auth.UserID != 1 && c.Auth.UserID != 2 {
		c.RespondWithErrorMessage(
			"only founders can see metrics",
			http.StatusForbidden,
		)
		return
	}

	metrics, status, err := models.GetMetrics()
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error fetching metrics: %s", err.Error()),
			status,
		)
		return
	}
	html := `<!DOCTYPE html>
<meta charset="utf-8">
<script type="text/javascript" src="https://www.google.com/jsapi"></script>`

	// Total Profiles
	idPrefix := `tp_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'Total Profiles', 'Customised Profiles', 'Total Sites', 'Engaged Sites'],`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`['%s',%d, %d, %d, %d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.TotalProfiles,
			metric.EditedProfiles,
			metric.TotalForums,
			metric.EngagedForums,
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'Profiles vs Sites (accumulative)',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {0: {logScale:false},
          1: {logScale:false}},
  series:{
  	0:{targetAxisIndex:0},
  	1:{targetAxisIndex:0},
  	2:{targetAxisIndex:1},
  	3:{targetAxisIndex:1}
  }
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>
<p style="width:900px">In the above chart a small divergence would indicate
linear growth of profiles per site = cubic growth of total population.</p>
<p style="width:900px">A massive divergence indicates a cubic growth of profiles
per site = an exponential growth of total population.</p>
`

	// Active + New Profiles
	idPrefix = `ap_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'New', 'Active (1 day)'],`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`['%s',%d,%d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.NewProfiles,
			metric.Signins,
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'New Profiles + Active Profiles',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {minValue: 0}
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>`

	// Guests vs Actives
	idPrefix = `gu_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'Unique Users', 'Active Profiles (1 day)'],`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`['%s',%d,%d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.Uniques,
			metric.Signins,
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'Unique Users + Active Profiles',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {minValue: 0}
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>`

	// Guests vs Actives
	idPrefix = `ga_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'Pageviews', 'Unique Visits'],`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`['%s',%d,%d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.Pageviews,
			metric.Visits,
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'Pageviews + Visits',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {minValue: 0}
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>`

	// Content Creation
	idPrefix = `cc_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'Comments', 'Conversations'],`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`['%s',%d,%d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.Comments,
			metric.Conversations,
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'Content Creation',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {minValue: 0}
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>`

	// Change in Content Creation
	idPrefix = `ccc_`
	html += `
<script type="text/javascript">
google.load("visualization", "1", {packages:["corechart"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
var ` + idPrefix + `data = google.visualization.arrayToDataTable([
  ['Date', 'Comments-Delta', 'Conversations-Delta'],`

	prev := metrics[0]
	for _, metric := range metrics[1:] {
		html += fmt.Sprintf(
			`['%s',%d,%d],`,
			metric.Timestamp.Local().Format("2006-01-02"),
			(metric.Comments - prev.Comments),
			(metric.Conversations - prev.Conversations),
		)
	}

	html += `]);

var ` + idPrefix + `options = {
  title: 'Content Creation',
  hAxis: {title: 'Date',  titleTextStyle: {color: '#333'}},
  vAxis: {minValue: 0}
};

var chart = new google.visualization.AreaChart(document.getElementById('` + idPrefix + `chart'));
chart.draw(` + idPrefix + `data, ` + idPrefix + `options);
}
</script>
<div id="` + idPrefix + `chart" style="width: 900px; height: 500px;"></div>`

	// Raw Data
	html += `<table>
<tr>
	<th>Timestamp</th>
	<th>Total forums</th>
	<th>Engaged forums</th>
	<th>Conversations</th>
	<th>Comments</th>
	<th>Total profiles</th>
	<th>Edited profiles</th>
	<th>New profiles</th>
	<th>Siginins</th>
	<th>Uniques</th>
	<th>Visits</th>
	<th>Pageviews</th>
</tr>
`

	for _, metric := range metrics {
		html += fmt.Sprintf(
			`<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>
`,
			metric.Timestamp.Local().Format("2006-01-02"),
			metric.TotalForums,
			metric.EngagedForums,
			metric.Conversations,
			metric.Comments,
			metric.TotalProfiles,
			metric.EditedProfiles,
			metric.NewProfiles,
			metric.Signins,
			metric.Uniques,
			metric.Visits,
			metric.Pageviews,
		)
	}
	html += `</table>`

	c.ResponseWriter.Header().Set("Content-Encoding", "text/html")
	c.WriteResponse([]byte(html), http.StatusOK)
}
