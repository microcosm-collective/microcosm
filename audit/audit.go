package audit

import (
	"net"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// Internal single-char indication of the auditable actions
const (
	create  = `C`
	replace = `R`
	update  = `U`
	delete  = `D`
)

// Create records an insert/create/POST action
func Create(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	profileID int64,
	seen time.Time,
	ipAddress net.IP) {

	recordAction(siteID, itemTypeID, itemID, profileID, seen, ipAddress, create)
}

// Replace records a full update/replace/PUT action
func Replace(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	profileID int64,
	seen time.Time,
	ipAddress net.IP) {

	recordAction(siteID, itemTypeID, itemID, profileID, seen, ipAddress, replace)
}

// Update records a partial update/PATCH action
func Update(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	profileID int64,
	seen time.Time,
	ipAddress net.IP) {

	recordAction(siteID, itemTypeID, itemID, profileID, seen, ipAddress, update)
}

// Delete records a remove/DELETE action
func Delete(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	profileID int64,
	seen time.Time,
	ipAddress net.IP) {

	recordAction(siteID, itemTypeID, itemID, profileID, seen, ipAddress, delete)
}

// recordAction actually appends to the audit log
func recordAction(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	profileID int64,
	seen time.Time,
	ipAddress net.IP,
	action string,
) {

	if ipAddress == nil {
		if glog.V(2) {
			glog.Infof("IP Address was nil for itemTypeId = %d", itemTypeID)
		}
		return
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	_, err = db.Exec(`INSERT INTO ips (
    site_id, item_type_id, item_id, profile_id, seen, action, ip
 ) VALUES (
    $1, $2, $3, $4, $5, $6, $7
 )`,
		siteID,
		itemTypeID,
		itemID,
		profileID,
		seen,
		action,
		ipAddress.String(),
	)
	if err != nil {
		glog.Error(err)
		return
	}
}
