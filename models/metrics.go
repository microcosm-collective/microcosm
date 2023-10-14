package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"

	conf "github.com/microcosm-cc/microcosm/config"
	h "github.com/microcosm-cc/microcosm/helpers"
)

var defaultLogoURL = "https://meta.microcosm.app/static/themes/1/logo.png"

// MetricType describes the attributes of all sites on Microcosm
type MetricType struct {
	Timestamp      time.Time
	Pageviews      int32
	Visits         int32
	Uniques        int32
	NewProfiles    int32
	EditedProfiles int32
	TotalProfiles  int32
	Signins        int32
	Comments       int32
	Conversations  int32
	EngagedForums  int32
	TotalForums    int32
}

// GetMetrics fetchs the latest metrics
func GetMetrics() ([]MetricType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return []MetricType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT job_timestamp
      ,pageviews
      ,visits
      ,uniques
      ,new_profiles
      ,edited_profiles
      ,total_profiles
      ,signins
      ,comments
      ,conversations
      ,engaged_forums
      ,total_forums
  FROM metrics
 WHERE job_timestamp IN (
           SELECT MAX(job_timestamp)
             FROM metrics
            GROUP BY job_timestamp::date
       )
 ORDER BY job_timestamp ASC`)
	if err != nil {
		return []MetricType{}, http.StatusInternalServerError, err
	}
	defer rows.Close()

	var ems []MetricType

	for rows.Next() {
		var metric MetricType
		err = rows.Scan(
			&metric.Timestamp,
			&metric.Pageviews,
			&metric.Visits,
			&metric.Uniques,
			&metric.NewProfiles,
			&metric.EditedProfiles,
			&metric.TotalProfiles,
			&metric.Signins,
			&metric.Comments,
			&metric.Conversations,
			&metric.EngagedForums,
			&metric.TotalForums,
		)
		if err != nil {
			return []MetricType{}, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}
		ems = append(ems, metric)
	}
	err = rows.Err()
	if err != nil {
		return []MetricType{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	return ems, http.StatusOK, nil
}

// UpdateMetrics builds a new set of metrics
func UpdateMetrics() error {
	if conf.ConfigStrings[conf.Environment] != `prod` {
		glog.Info("dev environment, skipping creation of metrics")
		return nil
	}

	// Subtract 24 hours from now (there is no Sub function that returns a time
	// value).
	end := time.Now()
	start := end.Add(-time.Hour * 24) // Will overlap/underlap on DST boundaries

	// Invoke metrics functions, store the results
	pCreated, pEdited, pTotal, err := ProfileMetrics(start)
	if err != nil {
		glog.Errorf("UpdateMetrics: %v", err.Error())
		return err
	}

	fTotal, fEngaged, err := ForumMetrics()
	if err != nil {
		glog.Errorf("UpdateMetrics: %v", err.Error())
		return err
	}

	signins, comments, convs, err := UserGenMetrics(start)
	if err != nil {
		glog.Errorf("UpdateMetrics: %v", err.Error())
		return err
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("UpdateMetrics: %v", err.Error())
		return err
	}

	_, err = db.Exec(`
INSERT INTO metrics(
	job_timestamp,
	new_profiles,
	edited_profiles,
	total_profiles,
	signins,

	comments,
	conversations,
	engaged_forums,
	total_forums
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,

	$6,
	$7,
	$8,
	$9
)`,
		end,
		pCreated,
		pEdited,
		pTotal,
		signins,

		comments,
		convs,
		fEngaged,
		fTotal,
	)
	if err != nil {
		glog.Errorf("UpdateMetrics: %+v", err)
		return err
	}

	return nil
}

// ProfileMetrics builds metrics about profiles
func ProfileMetrics(start time.Time) (created int, edited int, total int, err error) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("ProfileMetrics: %s", err.Error())
		return
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&total)
	if err != nil {
		return
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE created >= $1`,
		start,
	).Scan(
		&created,
	)
	if err != nil {
		return
	}

	err = db.QueryRow(`
SELECT COUNT(*)
  FROM profiles
 WHERE profile_name !~ 'user*'
   AND avatar_id IS NOT NULL`,
	).Scan(
		&edited,
	)

	return
}

// ForumMetrics builds metrics about sites
func ForumMetrics() (total int, engaged int, err error) {
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("ForumMetrics: %s", err.Error())
		return
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM sites`).Scan(&total)
	if err != nil {
		return
	}

	err = db.QueryRow(`
SELECT COUNT(*) FROM (
           SELECT s.site_id
             FROM sites s
                  JOIN microcosms m ON s.site_id = m.site_id
                  JOIN conversations c ON m.microcosm_id = c.microcosm_id
                  JOIN comments cm ON c.conversation_id = cm.item_id
            UNION
           SELECT s.site_id
             FROM sites s
                  JOIN microcosms m ON s.site_id = m.site_id
                  JOIN events e ON m.microcosm_id = e.microcosm_id
                  JOIN comments cm ON e.event_id = cm.item_id	
            WHERE s.logo_url !~ $1
              AND (
                      SELECT COUNT(*)
                        FROM profiles p
                       WHERE p.site_id = s.site_id
                  ) > 1
            GROUP BY s.site_id
       ) AS t`,
		defaultLogoURL,
	).Scan(
		&engaged,
	)

	return
}

// UserGenMetrics builds metrics about user activity
func UserGenMetrics(
	start time.Time,
) (
	signins int,
	comments int,
	convs int,
	err error,
) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("UserGenMetrics: %s", err.Error())
		return
	}

	err = db.QueryRow(
		`SELECT COUNT(*) FROM profiles where profiles.last_active > $1`,
		start,
	).Scan(
		&signins,
	)
	if err != nil {
		return
	}

	err = db.QueryRow(
		`SELECT COUNT(*) FROM comments WHERE created >= $1`,
		start,
	).Scan(
		&comments,
	)
	if err != nil {
		return
	}

	err = db.QueryRow(
		`SELECT COUNT(*) FROM conversations WHERE created >= $1`,
		start,
	).Scan(
		&convs,
	)
	return
}
