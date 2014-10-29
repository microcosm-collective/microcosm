package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

type PollsType struct {
	Polls h.ArrayType    `json:"polls"`
	Meta  h.CoreMetaType `json:"meta"`
}

type PollSummaryType struct {
	ItemSummary

	PollQuestion       string      `json:"question"`
	Multi              bool        `json:"multi"`
	PollOpen           bool        `json:"pollOpen"`
	VotingEndsNullable pq.NullTime `json:"-"`
	VotingEnds         string      `json:"pollCloses,omitempty"`

	ItemSummaryMeta
}

type PollType struct {
	ItemDetail

	// Type Specific
	PollQuestion string           `json:"question"`
	Multi        bool             `json:"multi"`
	PollOpen     bool             `json:"pollOpen"`
	Choices      []PollChoiceType `json:"choices,omitempty"`
	VoterCount   int64            `json:"voterCount"`

	// Type Specific Optional
	VotingEndsNullable pq.NullTime `json:"-"`
	VotingEnds         string      `json:"pollCloses,omitempty"`

	ItemDetailCommentsAndMeta
}

type PollChoiceType struct {
	Id         int64  `json:"id"`
	Choice     string `json:"choice"`
	Order      int64  `json:"order"`
	Votes      int64  `json:"votes"`
	VoterCount int64  `json:"voterCount"`
}

func (m *PollType) Validate(
	siteId int64,
	profileId int64,
	exists bool,
) (
	int,
	error,
) {

	m.Title = SanitiseText(m.Title)
	m.PollQuestion = SanitiseText(m.PollQuestion)

	// Does the Microcosm specified exist on this site?
	if !exists {
		_, status, err := GetMicrocosmSummary(siteId, m.MicrocosmId, profileId)
		if err != nil {
			return status, err
		}
	}

	if exists {
		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			return http.StatusBadRequest,
				errors.New("You must provide a reason for the update")
		} else {
			m.Meta.EditReason = ShoutToWhisper(m.Meta.EditReason)
		}
	}

	if m.MicrocosmId <= 0 {
		return http.StatusBadRequest,
			errors.New("You must specify a Microcosm ID")
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, errors.New("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	if strings.Trim(m.PollQuestion, " ") == "" {
		return http.StatusBadRequest,
			errors.New("You must supply a question that the poll will answer")
	}

	m.PollQuestion = ShoutToWhisper(m.PollQuestion)

	if m.Choices == nil || len(m.Choices) == 0 {
		return http.StatusBadRequest,
			errors.New("You must supply choices for the poll")
	} else {
		for ii := 0; ii < len(m.Choices); ii++ {
			m.Choices[ii].Choice = SanitiseText(m.Choices[ii].Choice)

			if strings.Trim(m.Choices[ii].Choice, " ") == "" {
				return http.StatusBadRequest,
					errors.New("Your poll choices must be populated")
			}

			m.Choices[ii].Choice = ShoutToWhisper(m.Choices[ii].Choice)
		}
	}

	if strings.Trim(m.VotingEnds, " ") != "" {
		votingEnds, err := time.Parse(time.RFC3339, m.VotingEnds)
		if err != nil {
			return http.StatusBadRequest, err
		} else {
			if !exists && votingEnds.Unix() < time.Now().Unix() {
				return http.StatusBadRequest,
					errors.New("Voting cannot close in the past")
			}

			m.VotingEndsNullable = pq.NullTime{Time: votingEnds, Valid: true}
		}
	}

	// Set defaults on child nodes
	for ii := 0; ii < len(m.Choices); ii++ {

		// Order is a sequence, we set the order to be the order that it was
		// posted to us.
		m.Choices[ii].Order = int64(ii) + 1
	}

	m.Meta.Flags.SetVisible()

	return http.StatusOK, nil
}

func (m *PollType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.Meta.EditedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	return http.StatusOK, nil
}

func (m *PollSummaryType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	return http.StatusOK, nil
}

func (m *PollType) Insert(siteId int64, profileId int64) (int, error) {

	status, err := m.Validate(siteId, profileId, false)
	if err != nil {
		return status, err
	}

	// Inputs are good, save to the database
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
INSERT INTO polls (
    microcosm_id, title, question, created, created_by,
    voting_ends, is_poll_open, is_multiple_choice
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8
) RETURNING poll_id`,
		m.MicrocosmId,
		m.Title,
		m.PollQuestion,
		m.Meta.Created,
		m.Meta.CreatedById,
		m.VotingEndsNullable,
		m.PollOpen,
		m.Multi,
	)

	var insertId int64
	err = row.Scan(&insertId)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error inserting data and returning ID: %v", err.Error()),
		)
	}
	m.Id = insertId

	for ii := 0; ii < len(m.Choices); ii++ {
		_, err := tx.Exec(`
INSERT INTO choices (
       poll_id, title, sequence
) VALUES (
       $1, $2, $3
)`,
			insertId,
			m.Choices[ii].Choice,
			m.Choices[ii].Order,
		)
		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Insert failed: %v", err.Error()),
			)
		}
	}

	err = IncrementMicrocosmItemCount(tx, m.MicrocosmId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypePoll], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func (m *PollType) Update(siteId int64, profileId int64) (int, error) {

	status, err := m.Validate(siteId, profileId, true)
	if err != nil {
		return status, err
	}

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	choiceIds := make([]string, len(m.Choices))
	seen := 0
	for ii := 0; ii < len(m.Choices); ii++ {
		if m.Choices[ii].Id > 0 {
			choiceIds[seen] = strconv.FormatInt(m.Choices[ii].Id, 10)
			seen++
		}
	}
	choiceIds = choiceIds[0:seen]

	if len(choiceIds) > 0 {

		_, err = tx.Exec(fmt.Sprintf(`
DELETE FROM choices
 WHERE poll_id = $1
   AND choice_id NOT IN (%s)`,
			strings.Join(choiceIds[0:seen], ","),
		),
			m.Id,
		)
		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Delete of choices failed: %v", err.Error()),
			)
		}
	}

	for ii := 0; ii < len(m.Choices); ii++ {
		if m.Choices[ii].Id > 0 {
			_, err = tx.Exec(`
UPDATE choices
   SET title = $1,
       sequence = $2,
       vote_count = $3
 WHERE choice_id = $4`,
				m.Choices[ii].Choice,
				m.Choices[ii].Order,
				m.Choices[ii].Votes,
				m.Choices[ii].Id,
			)
		} else {
			_, err = tx.Exec(`
INSERT INTO choices (
    poll_id, title, sequence
) VALUES (
    $1, $2, $3
)`,
				m.Id,
				m.Choices[ii].Choice,
				m.Choices[ii].Order,
			)
		}

		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Insert or Update of choices failed: %v", err.Error()),
			)
		}
	}

	_, err = tx.Exec(`
UPDATE polls
   SET microcosm_id = $2
      ,title = $3
      ,question = $4
      ,edited = $5
      ,edited_by = $6
      ,edit_reason = $7
      ,voting_ends = $8
      ,is_poll_open = $9
      ,is_multiple_choice = $10
 WHERE poll_id = $1`,
		m.Id,
		m.MicrocosmId,
		m.Title,
		m.PollQuestion,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
		m.VotingEndsNullable,
		m.PollOpen,
		m.Multi,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Update of poll failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypePoll], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func (m *PollType) Patch(ac AuthContext, patches []h.PatchType) (int, error) {

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {

		m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}
		m.Meta.EditedByNullable = sql.NullInt64{Int64: ac.ProfileId, Valid: true}

		var column string
		patch.ScanRawValue()
		switch patch.Path {
		case "/meta/flags/sticky":
			column = "is_sticky"
			m.Meta.Flags.Sticky = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set sticky to %t", m.Meta.Flags.Sticky)
		case "/meta/flags/open":
			column = "is_open"
			m.Meta.Flags.Open = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set open to %t", m.Meta.Flags.Open)
		case "/meta/flags/deleted":
			column = "is_deleted"
			m.Meta.Flags.Deleted = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set delete to %t", m.Meta.Flags.Deleted)
		case "/meta/flags/moderated":
			column = "is_moderated"
			m.Meta.Flags.Moderated = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set moderated to %t", m.Meta.Flags.Moderated)
		default:
			return http.StatusBadRequest,
				errors.New("Unsupported path in patch replace operation")
		}

		m.Meta.Flags.SetVisible()
		_, err = tx.Exec(`
UPDATE polls
   SET `+column+` = $2
      ,is_visible = $3
      ,edited = $4
      ,edited_by = $5
      ,edit_reason = $6
 WHERE poll_id = $1`,
			m.Id,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
			m.Meta.EditedNullable,
			m.Meta.EditedByNullable,
			m.Meta.EditReason,
		)
		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Update failed: %v", err.Error()),
			)
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func (m *PollType) Delete() (int, error) {

	// Delete resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE polls
   SET is_deleted = true
      ,is_visible = false
 WHERE poll_id = $1`,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	err = DecrementMicrocosmItemCount(tx, m.MicrocosmId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypePoll], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func GetPoll(siteId int64, id int64, profileId int64) (PollType, int, error) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcPollKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, PollType{}); ok {
		m := val.(PollType)
		m.FetchProfileSummaries(siteId)
		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return PollType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var m PollType
	err = db.QueryRow(`
SELECT p.poll_id
      ,p.microcosm_id
      ,p.title
      ,p.question
      ,p.created
      ,p.created_by
      ,p.edited
      ,p.edited_by
      ,p.edit_reason
      ,p.voter_count
      ,p.voting_ends
      ,p.is_sticky
      ,p.is_open
      ,p.is_deleted
      ,p.is_moderated
      ,p.is_visible
      ,p.is_poll_open
      ,p.is_multiple_choice
  FROM polls p
      ,microcosms m
 WHERE p.microcosm_id = m.microcosm_id
   AND m.site_id = $1
   AND m.is_deleted IS NOT TRUE
   AND m.is_moderated IS NOT TRUE
   AND p.poll_id = $2
   AND p.is_deleted IS NOT TRUE
   AND p.is_moderated IS NOT TRUE`,
		siteId,
		id,
	).Scan(
		&m.Id,
		&m.MicrocosmId,
		&m.Title,
		&m.PollQuestion,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.VoterCount,
		&m.VotingEndsNullable,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.PollOpen,
		&m.Multi,
	)
	if err == sql.ErrNoRows {
		return PollType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", id),
		)
	} else if err != nil {
		return PollType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.Meta.EditReasonNullable.Valid {
		m.Meta.EditReason = m.Meta.EditReasonNullable.String
	}
	if m.Meta.EditedNullable.Valid {
		m.Meta.Edited = m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
	}
	if m.VotingEndsNullable.Valid {
		m.VotingEnds = m.VotingEndsNullable.Time.Format(time.RFC3339Nano)
	}

	rows2, err := db.Query(`
SELECT choice_id,
       title,
       vote_count,
       voter_count,
       sequence
  FROM choices
 WHERE poll_id = $1
 ORDER BY sequence ASC`,
		m.Id,
	)
	if err != nil {
		return PollType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}
	defer rows2.Close()

	var choices []PollChoiceType

	for rows2.Next() {
		choice := PollChoiceType{}
		err = rows2.Scan(
			&choice.Id,
			&choice.Choice,
			&choice.Votes,
			&choice.VoterCount,
			&choice.Order,
		)
		choices = append(choices, choice)
		if err != nil {
			return PollType{}, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}
	}
	err = rows2.Err()
	if err != nil {
		return PollType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows2: %v", err.Error()),
		)
	}
	rows2.Close()

	m.Choices = choices
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypePoll, m.Id),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmId),
				h.ItemTypeMicrocosm,
				m.MicrocosmId,
			),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)
	return m, http.StatusOK, nil
}

func GetPollSummary(
	siteId int64,
	id int64,
	profileId int64,
) (
	PollSummaryType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcPollKeys[c.CacheSummary], id)
	if val, ok := c.CacheGet(mcKey, PollSummaryType{}); ok {
		m := val.(PollSummaryType)
		_, status, err := GetMicrocosmSummary(siteId, m.MicrocosmId, profileId)
		if err != nil {
			return PollSummaryType{}, status, err
		}
		m.FetchProfileSummaries(siteId)
		return m, 0, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return PollSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var m PollSummaryType
	err = db.QueryRow(`
SELECT poll_id
      ,microcosm_id
      ,title
      ,question
      ,created
      ,created_by
      ,voting_ends
      ,is_sticky
      ,is_open
      ,is_deleted
      ,is_moderated
      ,is_visible
      ,is_poll_open
      ,is_multiple_choice
      ,(SELECT COUNT(*) AS total_comments
          FROM flags
         WHERE parent_item_type_id = 7
           AND parent_item_id = $1
           AND item_is_deleted IS NOT TRUE
           AND item_is_moderated IS NOT TRUE) AS comment_count
      ,view_count
  FROM polls
 WHERE poll_id = $1
   AND is_deleted(7, poll_id) IS FALSE`,
		id,
	).Scan(
		&m.Id,
		&m.MicrocosmId,
		&m.Title,
		&m.PollQuestion,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.VotingEndsNullable,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.PollOpen,
		&m.Multi,
		&m.CommentCount,
		&m.ViewCount,
	)
	if err == sql.ErrNoRows {
		return PollSummaryType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", id),
		)
	} else if err != nil {
		return PollSummaryType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.VotingEndsNullable.Valid {
		m.VotingEnds = m.VotingEndsNullable.Time.Format(time.RFC3339Nano)
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypePoll, m.Id),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmId),
				h.ItemTypeMicrocosm,
				m.MicrocosmId,
			),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)
	return m, http.StatusOK, nil
}

func GetPolls(
	siteId int64,
	profileId int64,
	limit int64,
	offset int64,
) (
	[]PollSummaryType,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []PollSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,p.poll_id
  FROM polls p
      ,microcosms m
 WHERE p.microcosm_id = m.microcosm_id
   AND m.site_id = $1
   AND m.is_deleted IS NOT TRUE
   AND m.is_moderated IS NOT TRUE
   AND p.is_deleted IS NOT TRUE
   AND p.is_moderated IS NOT TRUE
 ORDER BY p.created ASC
 LIMIT $2
OFFSET $3`,
		siteId,
		limit,
		offset,
	)
	if err != nil {
		return []PollSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}
	defer rows.Close()

	var ems []PollSummaryType

	var total int64
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []PollSummaryType{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		m, status, err := GetPollSummary(siteId, id, profileId)
		if err != nil {
			return []PollSummaryType{}, 0, 0, status, err
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []PollSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []PollSummaryType{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("not enough records, "+
					"offset (%d) would return an empty page.", offset),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
