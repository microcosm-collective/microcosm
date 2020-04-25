package server

import (
	"github.com/microcosm-cc/microcosm/models"
)

// Field name   | Mandatory? | Allowed values  | Allowed special characters
// ----------   | ---------- | --------------  | --------------------------
// Seconds      | Yes        | 0-59            | * / , -
// Minutes      | Yes        | 0-59            | * / , -
// Hours        | Yes        | 0-23            | * / , -
// Day of month | Yes        | 1-31            | * / , - ?
// Month        | Yes        | 1-12 or JAN-DEC | * / , -
// Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?

var (
	jobs = map[string]func(){
		//SS MI HH  DOM MON DOW
		"  0  *     *    *   *   *": models.UpdateViewCounts,          // Every minute
		" 30  *     *    *   *   *": models.UpdateWhosOnline,          // Every minute at 30s
		"  0 30     *    *   *   *": models.UpdateAllSiteStats,        // Every hour at half past
		"  0  0     0/4  *   *   *": models.UpdateMetricsCron,         // Every day at midnight and every 4 hours thereafter
		"  0  0     1    *   *   *": models.UpdateMicrocosmItemCounts, // Every day at 1am
		"  0  0     4    *   *   *": models.DeleteOldUpdates,          // Every day at 4am
		"  0  0     5    *   *   *": models.DeleteOrphanedHuddles,     // Every day at 5am
		"  0  5     5    *   *   0": models.UpdateProfileCounts,       // Every Sunday at 5.05am
	}
)
