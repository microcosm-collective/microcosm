package resolver

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type vbq struct {
	key        string
	itemTypeID int64
}

var (
	vbqs = []vbq{
		{key: "f", itemTypeID: h.ItemTypes[h.ItemTypeMicrocosm]},
		{key: "forumid", itemTypeID: h.ItemTypes[h.ItemTypeMicrocosm]},
		{key: "p", itemTypeID: h.ItemTypes[h.ItemTypeComment]},
		{key: "pmid", itemTypeID: h.ItemTypes[h.ItemTypeHuddle]},
		{key: "postid", itemTypeID: h.ItemTypes[h.ItemTypeComment]},
		{key: "t", itemTypeID: h.ItemTypes[h.ItemTypeConversation]},
		{key: "threadid", itemTypeID: h.ItemTypes[h.ItemTypeConversation]},
		{key: "u", itemTypeID: h.ItemTypes[h.ItemTypeProfile]},
		{key: "userid", itemTypeID: h.ItemTypes[h.ItemTypeProfile]},
	}

	// yields forumid
	vbAnnouncement = regexp.MustCompile(`announcement([0-9]+).*$`)
	vbForum        = regexp.MustCompile(`forum([0-9]+)\.html$`)
	vbForumPage    = regexp.MustCompile(`forum([0-9]+)-([0-9]+)\.html$`)

	// yields thread
	vbThread          = regexp.MustCompile(`thread([0-9]+)\.html$`)
	vbThreadPage      = regexp.MustCompile(`thread([0-9]+)-([0-9]+)\.html$`)
	vbThreadPrint     = regexp.MustCompile(`printthread([0-9]+)\.html$`)
	vbThreadPrintPage = regexp.MustCompile(`printthread([0-9]+)-([0-9]+)\.html$`)

	// yields threadid to be translated to some post id
	vbLastPostInThread = regexp.MustCompile(`lastpostinthread([0-9]+)\.html$`)
	vbNewPostInThread  = regexp.MustCompile(`newpostinthread([0-9]+)\.html$`)

	// yields postid
	vbPost         = regexp.MustCompile(`post([0-9]+)\.html$`)
	vbPostPosition = regexp.MustCompile(`post([0-9]+)-[0-9]+\.html$`)

	// yields memberid
	vbMember = regexp.MustCompile(`member([0-9]+).*\.html$`)

	// yields attachmentid
	vbAttachment = regexp.MustCompile(`attachments/([0-9]+)d[0-9]+-.*$`)

	// Random URLS
	vbMemberList       = regexp.MustCompile(`memberslist/?$`)
	vbMemberListLetter = regexp.MustCompile(`memberslist/([0a-z])[0-9]+\.html$`)
	vbOnline           = regexp.MustCompile(`online.php$`)
	vbPMs              = regexp.MustCompile(`private.php$`)
	vbSubscription     = regexp.MustCompile(`subscription.php$`)
	vbUserCP           = regexp.MustCompile(`usercp.php$`)

	vbPostsPerThreadPage  int64 = 50
	vbThreadsPerForumPage int64 = 25
)

func resolveVbulletinURL(redirect Redirect, profileID int64) Redirect {

	// Query string checks are cheap, so we do those first
	qs := redirect.ParsedURL.Query()

	for _, q := range vbqs {
		i := atoi64(qs.Get(q.key))
		if i > 0 {
			i = getNewID(redirect.Origin.OriginID, q.itemTypeID, i)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}

			redirect.ItemTypeID = q.itemTypeID
			redirect.ItemID = i

			break
		}
	}

	i := atoi64(qs.Get("page"))
	if i > 0 {
		switch redirect.ItemTypeID {
		case h.ItemTypes[h.ItemTypeConversation]:
			redirect.Offset = pageToOffset(i, vbPostsPerThreadPage)
		case h.ItemTypes[h.ItemTypeMicrocosm]:
			redirect.Offset = pageToOffset(i, vbThreadsPerForumPage)
		default:
			redirect.Offset = pageToOffset(i, vbPostsPerThreadPage)
		}
	}

	action := qs.Get("goto")
	if action != "" {
		switch action {
		case "newpost":
			action = ActionNewComment
		default:
		}
	}

	// Move on to path based searches, but only if we haven't found anything.
	// These are potentially expensive, so these are ordered by the most likely
	// to the least likely... threads first, posts next, forums after, and then
	// the rest.

	// Look at the URL itself
	path := redirect.ParsedURL.Path

	// Thread redirects
	if redirect.ItemTypeID == 0 {
		matches := vbLastPostInThread.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Action = ActionNewComment
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbNewPostInThread.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Action = ActionNewComment
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbThreadPrintPage.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Offset = pageToOffset(atoi64(matches[2]), vbPostsPerThreadPage)
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbThreadPrint.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbThreadPage.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i

			// This is no longer done, we just send people to the first page
			// This is due to a massive increase in errors as it turned out there
			// were a lot of bad links in people's posts that vBulletin was more
			// forgiving about (it would auto-send people to the last page)
			//redirect.Offset = pageToOffset(atoi64(matches[2]), vbPostsPerThreadPage)
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbThread.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeConversation]

			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	// Comment redirects
	if redirect.ItemTypeID == 0 {
		matches := vbPostPosition.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeComment]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Action = ActionCommentInContext
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbPost.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeComment]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Action = ActionCommentInContext
		}
	}

	// Microcosm redirects
	if redirect.ItemTypeID == 0 {
		matches := vbForumPage.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeMicrocosm]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
			redirect.Offset = pageToOffset(atoi64(matches[2]), vbThreadsPerForumPage)
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbForum.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeMicrocosm]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbAnnouncement.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeMicrocosm]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	// Profile redirects
	if redirect.ItemTypeID == 0 {
		matches := vbMemberListLetter.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
			redirect.Action = ActionSearch
			redirect.Search = matches[1]
		}
	}

	if redirect.ItemTypeID == 0 {
		matches := vbMember.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	// Attachment redirects
	if redirect.ItemTypeID == 0 {
		matches := vbAttachment.FindStringSubmatch(path)
		if len(matches) > 0 {
			redirect.ItemTypeID = h.ItemTypes[h.ItemTypeAttachment]
			i = getNewID(
				redirect.Origin.OriginID,
				redirect.ItemTypeID,
				atoi64(matches[1]),
			)
			if i == 0 {
				redirect.Status = http.StatusNotFound
				return redirect
			}
			redirect.ItemID = i
		}
	}

	// Random URLs
	if redirect.ItemTypeID == 0 && vbMemberList.MatchString(path) {
		redirect.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
	}

	if redirect.ItemTypeID == 0 && vbOnline.MatchString(path) {
		redirect.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
		redirect.Action = ActionWhoIsOnline
	}

	if redirect.ItemTypeID == 0 && vbPMs.MatchString(path) {
		redirect.ItemTypeID = h.ItemTypes[h.ItemTypeHuddle]
	}

	if redirect.ItemTypeID == 0 && vbSubscription.MatchString(path) {
		redirect.ItemTypeID = h.ItemTypes[h.ItemTypeUpdate]
	}

	if redirect.ItemTypeID == 0 && vbUserCP.MatchString(path) {
		redirect.ItemTypeID = h.ItemTypes[h.ItemTypeUpdate]
	}

	// Construct the actual URLs to send people to
	canPaginate := false
	switch redirect.ItemTypeID {
	case h.ItemTypes[h.ItemTypeMicrocosm]:
		redirect.ItemType = h.ItemTypeMicrocosm

		if redirect.ItemID > 0 {
			redirect.URL.Href = fmt.Sprintf(
				"%s/%d",
				h.APITypeMicrocosm,
				redirect.ItemID,
			)
		} else {
			redirect.URL.Href = h.APITypeMicrocosm
		}
		redirect.URL.Rel = redirect.ItemType
		canPaginate = true

	case h.ItemTypes[h.ItemTypeConversation]:
		redirect.ItemType = h.ItemTypeConversation

		switch redirect.Action {
		case ActionNewComment:
			if redirect.ItemID > 0 {

				t, _, err := models.GetLastReadTime(
					h.ItemTypes[h.ItemTypeConversation],
					redirect.ItemID,
					profileID,
				)
				if err != nil {
					redirect.Status = http.StatusNotFound
					return redirect
				}

				commentID, _, err := models.GetNextOrLastCommentID(
					h.ItemTypes[h.ItemTypeConversation],
					redirect.ItemID,
					t,
					profileID,
				)
				if err != nil {
					redirect.Status = http.StatusNotFound
					return redirect
				}

				redirect.ItemTypeID = h.ItemTypes[h.ItemTypeComment]
				redirect.ItemType = h.ItemTypeComment
				redirect.ItemID = commentID

				redirect.URL.Href = fmt.Sprintf(
					"%s/%d",
					h.APITypeComment,
					redirect.ItemID,
				)
				redirect.URL.Rel = redirect.ItemType
				redirect.Action = ActionCommentInContext

			} else {
				redirect.Status = http.StatusNotFound
				return redirect
			}
		default:
			if redirect.ItemID > 0 {
				redirect.URL.Href = fmt.Sprintf(
					"%s/%d",
					h.APITypeConversation,
					redirect.ItemID,
				)
				redirect.URL.Rel = redirect.ItemType
				canPaginate = true
			} else {
				redirect.Status = http.StatusNotFound
				return redirect
			}
		}

	case h.ItemTypes[h.ItemTypeComment]:
		redirect.ItemType = h.ItemTypeComment

		if redirect.ItemID > 0 {
			redirect.URL.Href = fmt.Sprintf(
				"%s/%d",
				h.APITypeComment,
				redirect.ItemID,
			)
			redirect.URL.Rel = redirect.ItemType
		} else {
			redirect.Status = http.StatusNotFound
			return redirect
		}

	case h.ItemTypes[h.ItemTypeAttachment]:
		redirect.ItemType = h.ItemTypeAttachment

		db, err := h.GetConnection()
		if err != nil {
			redirect.Status = http.StatusNotFound
			return redirect
		}

		var fileSha1 string
		err = db.QueryRow(`--Get Attachment ID
SELECT file_sha1
  FROM attachments
 WHERE attachment_meta_id = $1`,
			redirect.ItemID,
		).Scan(&fileSha1)
		if err != nil {
			redirect.Status = http.StatusNotFound
			return redirect
		}

		site, _, err := models.GetSite(redirect.Origin.SiteID)
		if err != nil {
			redirect.Status = http.StatusNotFound
			return redirect
		}

		redirect.URL.Href = fmt.Sprintf(
			"https://%s.microcosm.app%s/%s",
			site.SubdomainKey,
			h.APITypeFile,
			fileSha1,
		)

	case h.ItemTypes[h.ItemTypeHuddle]:
		redirect.ItemType = h.ItemTypeHuddle

		if redirect.ItemID > 0 {
			redirect.URL.Href = fmt.Sprintf(
				"%s/%d",
				h.APITypeHuddle,
				redirect.ItemID,
			)
		} else {
			redirect.URL.Href = h.APITypeHuddle
		}
		redirect.URL.Rel = redirect.ItemType

	case h.ItemTypes[h.ItemTypeProfile]:
		redirect.ItemType = h.ItemTypeProfile

		if redirect.ItemID > 0 {
			redirect.URL.Href = fmt.Sprintf(
				"%s/%d",
				h.APITypeProfile,
				redirect.ItemID,
			)
		} else {
			switch redirect.Action {
			case ActionSearch:
				redirect.URL.Href = h.APITypeProfile + "?q=" + redirect.Search
			case ActionWhoIsOnline:
				redirect.URL.Href = h.APITypeProfile + "?online=true"
			default:
				redirect.URL.Href = h.APITypeProfile
			}
			canPaginate = true
		}
		redirect.URL.Rel = redirect.ItemType

	case h.ItemTypes[h.ItemTypeUpdate]:
		redirect.ItemType = h.ItemTypeUpdate
		redirect.URL.Href = h.APITypeUpdate
		redirect.URL.Rel = redirect.ItemType

	default:
		redirect.Status = http.StatusNotFound
		return redirect
	}

	// Append offset if applicable
	if canPaginate && redirect.Offset > 0 {
		url, err := url.Parse(redirect.URL.Href)
		if err != nil {
			redirect.Status = http.StatusNotFound
			return redirect
		}

		q := url.Query()
		q.Add("offset", strconv.FormatInt(redirect.Offset, 10))
		url.RawQuery = q.Encode()

		redirect.URL.Href = url.String()
	} else {
		redirect.Offset = 0
	}

	redirect.Status = http.StatusMovedPermanently

	return redirect
}
