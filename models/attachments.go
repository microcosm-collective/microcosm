package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// AttachmentsType is a collection of attachments
type AttachmentsType struct {
	Attachments h.ArrayType    `json:"attachments"`
	Meta        h.CoreMetaType `json:"meta"`
}

// AttachmentType describes an attachment
type AttachmentType struct {
	AttachmentID     int64          `json:"-"`
	ProfileID        int64          `json:"profileId"`
	AttachmentMetaID int64          `json:"-"`
	ItemTypeID       int64          `json:"-"`
	FileHash         string         `json:"fileHash"`
	FileName         string         `json:"fileName"`
	FileExt          string         `json:"fileExt"`
	ItemID           int64          `json:"-"`
	Created          time.Time      `json:"created"`
	ViewCount        int64          `json:"-"`
	Meta             h.CoreMetaType `json:"meta"`
}

// Import saves an attachment
func (m *AttachmentType) Import() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("Transaction failed")
	}

	return http.StatusOK, nil
}

// Insert saves an attachment
func (m *AttachmentType) Insert() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("Transaction failed")
	}

	return http.StatusOK, nil
}

func (m *AttachmentType) insert(q h.Er) (int, error) {
	fileNameBits := strings.Split(m.FileName, ".")
	m.FileExt = "unk"
	if len(fileNameBits) > 0 {
		m.FileExt = fileNameBits[len(fileNameBits)-1]
	}

	var insertID int64
	err := q.QueryRow(`
INSERT INTO attachments (
	profile_id, attachment_meta_id, item_type_id, file_sha1, item_id,
	created, view_count, file_name, file_ext
) VALUES (
	$1, $2, $3, $4, $5,
	$6, $7, $8, $9
) RETURNING attachment_id`,
		m.ProfileID,
		m.AttachmentMetaID,
		m.ItemTypeID,
		m.FileHash,
		m.ItemID,
		m.Created,
		m.ViewCount,
		m.FileName,
		m.FileExt,
	).Scan(
		&insertID,
	)
	if err != nil {
		glog.Errorf(
			"q.QueryRow(%d, %d, %d, %s, %d, %v, %d, %s, %s).Scan() %+v",
			m.ProfileID,
			m.AttachmentMetaID,
			m.ItemTypeID,
			m.FileHash,
			m.ItemID,
			m.Created,
			m.ViewCount,
			m.FileName,
			m.FileExt,
			err,
		)
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID")
	}
	m.AttachmentID = insertID

	go PurgeCache(m.ItemTypeID, m.ItemID)

	return http.StatusOK, nil
}

// Update saves the changes to an attachment
func (m *AttachmentType) Update() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE attachments
   SET created = $1
 WHERE attachment_id = $2`,
		m.Created,
		m.AttachmentID,
	)
	if err != nil {
		glog.Errorf("tx.Exec(%v, %d) %+v", m.Created, m.AttachmentID, err)
		return http.StatusInternalServerError,
			fmt.Errorf("Attachment update failed")
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("Transaction failed")
	}

	return http.StatusOK, nil
}

// GetAttachment returns an attachment
func GetAttachment(
	itemTypeID int64,
	itemID int64,
	fileHash string,
	latest bool,
) (
	AttachmentType,
	int,
	error,
) {
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return AttachmentType{}, http.StatusInternalServerError, err
	}

	// filehash is optional
	var where string
	if fileHash != `` {
		where = `item_type_id = $1 AND item_id = $2 AND file_sha1 = $3`
	} else {
		where = `item_type_id = $1 AND item_id = $2`
	}

	// fetch the last created attachment
	var order string
	if latest {
		order = `ORDER BY created DESC LIMIT 1`
	}

	sqlQuery := fmt.Sprintf(`
SELECT attachment_id
      ,profile_id
      ,attachment_meta_id
      ,item_type_id
      ,file_sha1
      ,item_id
      ,created
      ,view_count
      ,file_name
      ,file_ext
  FROM attachments
 WHERE %s
 %s`, where, order)

	var rows *sql.Rows
	if fileHash != `` {
		rows, err = db.Query(sqlQuery, itemTypeID, itemID, fileHash)
		if err != nil {
			glog.Errorf(
				"db.Query(%d, %d, `%s`) %+v",
				itemTypeID,
				itemID,
				fileHash,
				err,
			)
			return AttachmentType{}, http.StatusInternalServerError,
				fmt.Errorf("Database query failed")
		}
	} else {
		rows, err = db.Query(sqlQuery, itemTypeID, itemID)
		if err != nil {
			glog.Errorf("db.Query(%d, %d) %+v", itemTypeID, itemID, err)
			return AttachmentType{}, http.StatusInternalServerError,
				fmt.Errorf("Database query failed")
		}
	}
	defer rows.Close()

	var m AttachmentType
	for rows.Next() {
		err = rows.Scan(
			&m.AttachmentID,
			&m.ProfileID,
			&m.AttachmentMetaID,
			&m.ItemTypeID,
			&m.FileHash,
			&m.ItemID,
			&m.Created,
			&m.ViewCount,
			&m.FileName,
			&m.FileExt,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return AttachmentType{}, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return AttachmentType{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	if m.AttachmentID == 0 {
		glog.Infof("m.AttachmentId == 0 for hash %s", fileHash)
		return AttachmentType{},
			http.StatusNotFound,
			fmt.Errorf("Resource not found")
	}

	return m, http.StatusOK, nil
}

// DeleteAttachment removes an attachment
func DeleteAttachment(
	itemTypeID int64,
	itemID int64,
	fileHash string,
) (
	int,
	error,
) {
	// TODO(matt): reset attach_count by cron

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if itemTypeID == h.ItemTypes[h.ItemTypeProfile] {

		var total int64

		err = tx.QueryRow(`
SELECT COUNT(*)
  FROM attachments
 WHERE item_type_id = $1
   AND item_id = $2`,
			itemTypeID,
			itemID,
		).Scan(
			&total,
		)
		if err != nil {
			glog.Errorf(
				"tx.QueryRow(%d, %d).Scan() %+v",
				itemTypeID,
				itemID,
				err,
			)
			return http.StatusInternalServerError,
				fmt.Errorf("Error fetching row")
		}

		if total <= 1 {
			glog.Infoln("total <= 1")
			return http.StatusInternalServerError,
				fmt.Errorf("Can not delete: only one avatar remaining")
		}

		//if active avatar, set to previous
		var location string

		err = tx.QueryRow(`
SELECT avatar_url
  FROM profiles
 WHERE profile_id = $1`,
			itemID,
		).Scan(
			&location,
		)
		if err != nil {
			glog.Errorf("tx.QueryRow(%d).Scan() %+v", itemID, err)
			return http.StatusInternalServerError,
				fmt.Errorf("Error fetching row")
		}

		if strings.HasPrefix(
			location,
			fmt.Sprintf("%s/%s", h.APITypeFile, fileHash),
		) {

			_, err = tx.Exec(`
UPDATE profiles
   SET avatar_url = $1 || file_sha1
      ,avatar_id = attachment_id
  FROM (
        SELECT file_sha1
              ,attachment_id
          FROM attachments
         WHERE item_type_id = $2
           AND item_id = $3
           AND file_sha1 NOT LIKE $4
         ORDER BY created DESC
         LIMIT 1
       ) att
 WHERE profile_id = $3`,
				fmt.Sprintf("%s/", h.APITypeFile),
				itemTypeID,
				itemID,
				fileHash,
			)
			if err != nil {
				glog.Errorf(
					"tx.Exec(`%s`, %d, %d, `%s`) %+v",
					fmt.Sprintf("%s/", h.APITypeFile),
					itemTypeID,
					itemID,
					fileHash,
					err,
				)
				return http.StatusInternalServerError,
					fmt.Errorf("Reassignment of avatar failed")
			}
		}
	}

	_, err = tx.Exec(`
DELETE FROM attachments
 WHERE item_type_id = $1
   AND item_id = $2
   AND file_sha1 = $3`,
		itemTypeID,
		itemID,
		fileHash,
	)
	if err != nil {
		glog.Errorf(
			"tx.Exec(%d, %d, %s) %+v",
			itemTypeID,
			itemID,
			fileHash,
			err,
		)
		return http.StatusInternalServerError, fmt.Errorf("Delete failed")
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("Transaction failed")
	}

	go PurgeCache(itemTypeID, itemID)

	return http.StatusOK, nil
}

// GetAttachments returns a collection of attachments
func GetAttachments(
	itemTypeID int64,
	itemID int64,
	limit int64,
	offset int64,
) (
	[]AttachmentType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []AttachmentType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() as total
      ,profile_id
      ,attachment_meta_id
      ,item_type_id
      ,file_sha1
      ,item_id
      ,created
      ,view_count
      ,file_name
      ,file_ext
  FROM attachments
 WHERE item_type_id = $1
   AND item_id = $2
 ORDER BY attachment_id
 LIMIT $3
OFFSET $4`,
		itemTypeID,
		itemID,
		limit,
		offset,
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			itemTypeID,
			itemID,
			limit,
			offset,
			err,
		)
		return []AttachmentType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	var total int64
	attachments := []AttachmentType{}
	for rows.Next() {
		m := AttachmentType{}
		err = rows.Scan(
			&total,
			&m.ProfileID,
			&m.AttachmentMetaID,
			&m.ItemTypeID,
			&m.FileHash,
			&m.ItemID,
			&m.Created,
			&m.ViewCount,
			&m.FileName,
			&m.FileExt,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []AttachmentType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		// TODO: add link to the file metadata and describe the
		// content-type of the file
		filePath := m.FileHash
		if m.FileExt != "" {
			filePath += `.` + m.FileExt
		}
		link := h.LinkType{
			Rel:   "related",
			Href:  fmt.Sprintf("%s/%s", h.APITypeFile, filePath),
			Title: "File resource",
		}
		m.Meta.Links = append(m.Meta.Links, link)
		attachments = append(attachments, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []AttachmentType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []AttachmentType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return attachments, total, pages, http.StatusOK, nil
}
