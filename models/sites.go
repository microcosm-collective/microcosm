package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/golang/glog"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	DefaultThemeId            int64  = 1
	DefaultBackgroundColor    string = `#FFFFFF`
	DefaultBackgroundPosition string = `tile`
	DefaultLinkColor          string = `#4082C3`
)

var DisallowedSubdomains = []string{
	"about",         // Passing off
	"abuse",         // Customer support
	"account",       // Passing off
	"accounts",      // Passing off
	"admin",         // Web panel
	"admins",        // Web panel
	"adminstrator",  // Web panel
	"adminstrators", // Web panel
	"api",           // Developer resources
	"assets",        // Developer resources
	"atom",          // Publishing
	"billing",       // Passing off
	"billings",      // Passing off
	"blog",          // Standard
	"bugs",          // Developer resources
	"calendar",      // Google Calendar
	"chat",          // Passing off
	"code",          // Developer resources
	"communities",   // Passing off
	"community",     // Passing off
	"contact",       // Passing off
	"contributors",  // Developer resources
	"coppa",         // Legal
	"copyright",     // Legal
	"css",           // Developer resources
	"customise",     // Customer support
	"customize",     // Customer support
	"demo",          // Microcosm Demo
	"dev",           // Environments
	"developer",     // Developer resources
	"developers",    // Developer resources
	"development",   // Environments
	"direct",        // CloudFlare DNS
	"docs",          // Google Docs
	"email",         // Email
	"example",       // Developer resources
	"feedback",      // Passing off
	"files",         // Web Dav
	"forum",         // Passing off
	"ftp",           // Files
	"git",           // Source control
	"help",          // Customer support
	"hostmaster",    // Email
	"imap",          // Email
	"inbox",         // Email
	"jabber",        // Google Chat
	"lab",           // Developer resources
	"login",         // Security
	"mail",          // Email
	"manage",        // Web panel
	"mobile",        // Standard
	"mx",            // DNS
	"official",      // Passing off
	"owa",           // Outlook Web Access
	"pages",         // Google Sites
	"payment",       // Passing off
	"policy",        //Legal
	"pop",           // Email
	"postmaster",    // Email
	"press",         // Passing off
	"privacy",       // Legal
	"prod",          // Environments
	"production",    // Environments
	"profile",       // Features
	"search",        // Features
	"secure",        // SSL
	"signup",        // Microcosm Registration
	"sitemap",       // Developer resources
	"sites",         // Google Sites
	"smtp",          // Email
	"ssl",           // SSL
	"staff",         // Passing off
	"stage",         // Environments
	"staging",       // Environments
	"static",        // Developer resources
	"status",        // Status updates
	"support",       // Customer support
	"username",      // Security
	"usernames",     // Security
	"users",         // Security
	"webmail",       // Email
	"webmaster",     // Email
	"www",           // Standard
	"xmpp",          // Google Chat
}

type SitesType struct {
	Sites h.ArrayType    `json:"sites"`
	Meta  h.CoreMetaType `json:"meta"`
}

type SiteType struct {
	Id                      int64          `json:"siteId"`
	Title                   string         `json:"title"`
	Description             string         `json:"description"`
	SubdomainKey            string         `json:"subdomainKey"`
	Domain                  string         `json:"domain"`
	DomainNullable          sql.NullString `json:"-"`
	OwnedById               int64          `json:"-"`
	OwnedBy                 interface{}    `json:"ownedBy"`
	ThemeId                 int64          `json:"themeId"`
	LogoUrl                 string         `json:"logoUrl"`
	FaviconUrl              string         `json:"faviconUrl,omitempty"`
	BackgroundColor         string         `json:"backgroundColor"`
	BackgroundUrl           string         `json:"backgroundUrl,omitempty"`
	BackgroundPosition      string         `json:"backgroundPosition,omitempty"`
	LinkColor               string         `json:"linkColor"`
	GaWebPropertyId         string         `json:"gaWebPropertyId,omitempty"`
	GaWebPropertyIdNullable sql.NullString `json:"-"`
	Menu                    []h.LinkType   `json:"menu"`

	Meta struct {
		h.CreatedType
		h.EditedType

		Flags struct {
			Deleted bool `json:"deleted"`
		} `json:"flags,omitempty"`

		h.CoreMetaType
	} `json:"meta"`
}

type SiteStatType struct {
	ActiveProfiles int64
	OnlineProfiles int64
	TotalProfiles  int64
	TotalConvs     int64
	TotalEvents    int64
	TotalComments  int64
}

type SiteHealthType struct {
	Site                SiteType            `json:"site"`
	DomainHealth        SiteHealthAttribute `json:"domainHealth"`
	BackgroundUrlHealth SiteHealthAttribute `json:"backgroundUrlHealth"`
	LogoUrlHealth       SiteHealthAttribute `json:"logoUrlHealth"`
	AnalyticsIDHealth   SiteHealthAttribute `json:"analyticsIDHealth"`
}

type SiteHealthAttribute struct {
	Set   bool        `json:"set"`
	Valid bool        `json:"valid"`
	Error string      `json:"error"`
	Value interface{} `json:"value"`
}

var regAlphaNum = regexp.MustCompile(`[A-Za-z0-9]+`)

func (m *SiteType) Validate(exists bool) (int, error) {

	m.Title = SanitiseText(m.Title)
	m.Description = SanitiseText(m.Description)
	m.SubdomainKey = SanitiseText(m.SubdomainKey)
	m.Domain = SanitiseText(m.Domain)

	if exists {
		if m.Id < 1 {
			return http.StatusBadRequest, errors.New("Invalid site ID")
		}
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest,
			errors.New("You must specify a site title")
	}

	if strings.Trim(m.Description, " ") == "" {
		return http.StatusBadRequest,
			errors.New("You must specify a site description")
	}

	if strings.Trim(m.SubdomainKey, " ") == "" {
		return http.StatusBadRequest,
			errors.New("You must specify a subdomain key")
	}

	if m.SubdomainKey != "" {
		for _, subdomain := range DisallowedSubdomains {
			if m.SubdomainKey == subdomain {
				return http.StatusBadRequest, errors.New(
					fmt.Sprintf(
						"Subdomain '%s' is reserved and cannot be used",
						m.SubdomainKey,
					),
				)
			}
		}
		if !regAlphaNum.MatchString(m.SubdomainKey) {
			return http.StatusBadRequest,
				errors.New("Subdomain key must be alphanumeric")
		}
	}

	m.BackgroundColor = strings.Trim(m.BackgroundColor, " ")
	if m.BackgroundColor == "" {
		m.BackgroundColor = DefaultBackgroundColor
	}

	if !h.IsValidColor(m.BackgroundColor) {
		return http.StatusBadRequest,
			errors.New(
				"Background color is not a valid HTML color (hex or named)",
			)
	}

	m.LinkColor = strings.Trim(m.LinkColor, " ")
	if m.LinkColor == "" {
		m.LinkColor = DefaultLinkColor
	}

	if !h.IsValidColor(m.LinkColor) {
		return http.StatusBadRequest,
			errors.New("Link color is not a valid HTML color (hex or named)")
	}

	validBackgroundPosition := map[string]bool{
		"cover":  true,
		"left":   true,
		"center": true,
		"right":  true,
		"tiled":  true,
	}

	m.BackgroundPosition = strings.Trim(strings.ToLower(m.BackgroundPosition), " ")
	if !validBackgroundPosition[m.BackgroundPosition] {
		m.BackgroundPosition = DefaultBackgroundPosition
	}

	if m.GaWebPropertyId != "" {
		if !strings.HasPrefix(m.GaWebPropertyId, "UA-") {
			return http.StatusBadRequest,
				errors.New(
					"gaWebPropertyId must be in the form of the UA-XXXX-Y " +
						"property ID that Google Analytics provided to you",
				)
		}
		m.GaWebPropertyIdNullable = sql.NullString{
			String: m.GaWebPropertyId,
			Valid:  true,
		}
	}

	return http.StatusOK, nil
}

func (m *SiteType) FetchProfileSummaries() (int, error) {

	profile, status, err := GetProfileSummary(m.Id, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	profile, status, err = GetProfileSummary(m.Id, m.OwnedById)
	if err != nil {
		return status, err
	}
	m.OwnedBy = profile

	// Site stats are also pulled from cache so that the site itself can be
	// cached for a long time, but the stats can be updated more frequently.
	// These are updated by eviction from a cron job, so have a long TTL in case
	// the cron job fails
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheCounts], m.Id)
	if val, ok := c.CacheGet(mcKey, []h.StatType{}); ok {
		m.Meta.Stats = val.([]h.StatType)
	} else {
		stats, err := GetSiteStats(m.Id)
		if err != nil {
			glog.Error(err)
		} else {
			m.Meta.Stats = stats
			c.CacheSet(mcKey, m.Meta.Stats, mcTtl)
		}
	}

	return http.StatusOK, nil
}

// IsReservedSubdomain checks if a subdomain is reserved or used by an existing
// site.
func IsReservedSubdomain(query string) (bool, error) {

	for _, subdomain := range DisallowedSubdomains {
		if subdomain == query {
			return true, nil
		}
	}

	_, status, err := GetSiteBySubdomain(query)
	if err == nil {
		return true, nil
	}
	if err != nil && status != http.StatusNotFound {
		return false,
			errors.New(
				fmt.Sprintf("Error fetching site by subdomain: %+v", err),
			)
	}

	return false, nil
}

// CreateOwnedSite creates a new site and a profile to own the site, based on
// the user details provided.
func CreateOwnedSite(
	site SiteType,
	user UserType,
) (
	SiteType,
	ProfileType,
	int,
	error,
) {

	status, err := site.Validate(false)
	if err != nil {
		return SiteType{}, ProfileType{}, status, err
	}

	// Create stub profile to serve as site owner
	profile := ProfileType{}
	profile.ProfileName = SuggestProfileName(user)
	profile.UserId = user.ID
	profile.Visible = true

	tx, err := h.GetTransaction()
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Could not start transaction: %v", err.Error()),
			)
	}
	defer tx.Rollback()

	// Create a site and owner profile in a single transaction
	rows, err := tx.Query(`
SELECT new_ids.new_site_id,
       new_ids.new_profile_id
  FROM create_owned_site(
           $1, $2, $3, $4, $5,
           $6, $7, $8, $9, $10,
           $11, $12, $13, $14
       ) AS new_ids`,
		site.Title,
		site.SubdomainKey,
		site.ThemeId,
		user.ID,
		profile.ProfileName,

		profile.AvatarIdNullable,
		profile.AvatarUrlNullable,
		site.DomainNullable,
		site.Description,
		site.LogoUrl,

		site.BackgroundUrl,
		site.BackgroundPosition,
		site.BackgroundColor,
		site.LinkColor,
	)
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error executing query: %v", err.Error()),
			)
	}
	defer rows.Close()

	var siteId int64
	var profileId int64
	for rows.Next() {
		err = rows.Scan(&siteId, &profileId)
		if err != nil {
			return SiteType{}, ProfileType{}, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf(
						"Error inserting data and returning IDs: %v",
						err.Error(),
					),
				)
		}
	}
	if rows.Err() != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf(
					"Error inserting data and returning IDs: %v",
					rows.Err().Error(),
				),
			)
	}
	rows.Close()

	site.Id = siteId
	profile.SiteId = siteId
	profile.Id = profileId

	// Create profile_options record for the newly created profile
	profileOptions, _, err := GetProfileOptionsDefaults(site.Id)
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf(
					"Could not load default profile options: %v",
					err.Error(),
				),
			)
	}
	profileOptions.ProfileId = profile.Id

	status, err = profileOptions.Insert(tx)
	if err != nil {
		return SiteType{}, ProfileType{}, status, errors.New(
			fmt.Sprintf(
				"Could not insert new profile options: %v",
				err.Error(),
			),
		)
	}

	err = tx.Commit()
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Transaction failed: %v", err.Error()),
			)
	}

	// Create attachment for avatar and attach it to profile
	gravatarUrl := MakeGravatarUrl(profile.ProfileName)
	fm, _, err := StoreGravatar(gravatarUrl)
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Could not store gravatar for profile: %+v", err),
			)
	}

	// Attach avatar to profile
	attachment, status, err := AttachAvatar(profile.Id, fm)
	if err != nil {
		return SiteType{}, ProfileType{}, status, errors.New(
			fmt.Sprintf("Could not attach avatar to profile: %v", err.Error()),
		)
	}

	// Construct URL to avatar, update profile with Avatar ID and URL
	filePath := fm.FileHash
	if fm.FileExt != "" {
		filePath += `.` + fm.FileExt
	}
	profile.AvatarUrlNullable = sql.NullString{
		String: fmt.Sprintf("%s/%s", h.ApiTypeFile, filePath),
		Valid:  true,
	}
	profile.AvatarIdNullable = sql.NullInt64{
		Int64: attachment.AttachmentId,
		Valid: true,
	}
	status, err = profile.Update()
	if err != nil {
		return SiteType{}, ProfileType{}, status, errors.New(
			fmt.Sprintf("Could not update profile with avatar: %v", err.Error()),
		)
	}

	email := EmailType{}
	email.From = "operations@microcosm.cc"
	email.ReplyTo = user.Email
	email.To = "founders@microcosm.cc"
	email.Subject = "New site created: " + site.Title
	email.BodyText = fmt.Sprintf(
		`Title: %s
Url: http://%s.microco.sm/
Email: %s`, site.Title, site.SubdomainKey, user.Email)
	email.BodyHTML = fmt.Sprintf(`<p>Title: %s</p>
<p>Url: <a href="http://%s.microco.sm/">http://%s.microco.sm/</a></p>
<p>Description: %s</p>
<p>Email: %s</p>`,
		site.Title,
		site.SubdomainKey,
		site.SubdomainKey,
		site.Description,
		user.Email,
	)
	email.Send(site.Id)

	return site, profile, http.StatusOK, nil
}

func (m *SiteType) Update() (int, error) {

	status, err := m.Validate(true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE sites
   SET title = $2
      ,description = $3
      ,domain = $4
      ,theme_id = $5
      ,logo_url = $6
      ,favicon_url = $7

      ,background_url = $8
      ,background_color = $9
      ,background_position = $10
      ,link_color = $11
      ,ga_web_property_id = $12

      ,is_deleted = $13
 WHERE site_id = $1`,
		m.Id,

		m.Title,
		m.Description,
		m.DomainNullable,
		m.ThemeId,
		m.LogoUrl,
		m.FaviconUrl,

		m.BackgroundUrl,
		m.BackgroundColor,
		m.BackgroundPosition,
		m.LinkColor,
		m.GaWebPropertyIdNullable,

		m.Meta.Flags.Deleted,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Update of site failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeSite], m.Id)
	return http.StatusOK, nil
}

func (m *SiteType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM sites
 WHERE site_id = $1`,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeSite], m.Id)

	return http.StatusOK, nil
}

func (m *SiteType) GetUrl() string {
	if m.Domain == "" {
		return "https://" + m.SubdomainKey + ".microco.sm"
	} else {
		return "http://" + m.Domain
	}
}

func GetSiteTitle(id int64) string {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheTitle], id)
	if val, ok := c.CacheGetString(mcKey); ok {
		return val
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return ""
	}

	title := ""
	err = db.QueryRow(`
SELECT title
  FROM sites
 WHERE site_id = $1`,
		id,
	).Scan(
		&title,
	)
	if err != nil {
		return ""
	}

	// Update cache
	c.CacheSetString(mcKey, title, mcTtl)

	return title
}

func GetSite(id int64) (SiteType, int, error) {

	// Try cache
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, SiteType{}); ok {
		m := val.(SiteType)
		// Site now caches the profile summaries as the benefits on performance
		// are too great to do this every time as context.MakeContext for every
		// controller uses a Site object
		// m.FetchProfileSummaries()
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return SiteType{}, http.StatusInternalServerError, err
	}

	var m SiteType
	err = db.QueryRow(`
SELECT s.site_id
      ,s.title
      ,s.description
      ,s.subdomain_key
      ,s.domain
      ,s.created
      ,s.created_by
      ,s.owned_by
      ,t.theme_id as theme_id
      ,CASE WHEN s.logo_url IS NOT NULL AND s.logo_url <> ''
            THEN s.logo_url
            ELSE t.logo_url
        END AS logo_url
      ,CASE WHEN s.background_url IS NOT NULL
            THEN s.background_url
            ELSE t.background_url
        END AS background_url
      ,CASE WHEN s.favicon_url IS NOT NULL
            THEN s.favicon_url
            ELSE t.favicon_url
        END AS favicon_url
      ,s.background_color
      ,s.background_position
      ,s.link_color
      ,ga_web_property_id
      ,is_deleted
  FROM sites s
      ,themes t
 WHERE s.theme_id = t.theme_id
   AND s.site_id = $1`,
		id,
	).Scan(
		&m.Id,
		&m.Title,
		&m.Description,
		&m.SubdomainKey,
		&m.DomainNullable,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.OwnedById,
		&m.ThemeId,
		&m.LogoUrl,
		&m.BackgroundUrl,
		&m.FaviconUrl,
		&m.BackgroundColor,
		&m.BackgroundPosition,
		&m.LinkColor,
		&m.GaWebPropertyIdNullable,
		&m.Meta.Flags.Deleted,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with site ID %d not found", id),
		)
	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.DomainNullable.Valid {
		m.Domain = m.DomainNullable.String
	}
	if m.GaWebPropertyIdNullable.Valid {
		m.GaWebPropertyId = m.GaWebPropertyIdNullable.String
	}
	menu, status, err := GetMenu(m.Id)
	if err != nil {
		return SiteType{}, status, errors.New(
			fmt.Sprintf("Error fetching menu: %v", err.Error()),
		)
	}
	m.Menu = menu
	if m.BackgroundUrl == "" {
		m.BackgroundPosition = ""
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeSite, m.Id),
			h.GetLink("microcosm", "", h.ItemTypeMicrocosm, 0),
			h.GetLink("profile", "", h.ItemTypeProfile, 0),
			h.LinkType{Rel: "legal", Href: "/api/v1/legal"},
		}
	m.FetchProfileSummaries()

	c.CacheSet(mcKey, m, mcTtl)

	return m, http.StatusOK, nil
}

// Calculate site statistics. This is expensive and should not
// be run to synchronously service a request.
func CalcSiteStats(siteId int64) (SiteStatType, error) {

	var stats SiteStatType
	db, err := h.GetConnection()
	if err != nil {
		return stats, err
	}

	// Active Profiles
	err = db.QueryRow(`
SELECT COUNT(*)
  FROM profiles
 WHERE site_id = $1
   AND last_active > current_date - integer '90'`,
		siteId,
	).Scan(
		&stats.ActiveProfiles,
	)
	if err != nil {
		return stats, err
	}

	// Online profiles
	err = db.QueryRow(`
SELECT COUNT(*)
  FROM profiles
 WHERE site_id = $1 
   AND last_active > NOW() - interval '90 minute'`,
		siteId,
	).Scan(
		&stats.OnlineProfiles,
	)
	if err != nil {
		return stats, err
	}

	// Totals
	err = db.QueryRow(`--CalcSiteStats
SELECT SUM(profiles) AS profiles
      ,SUM(conversations) AS conversations
      ,SUM(events) AS events
      ,SUM(comments) AS comments
  FROM (
    SELECT 0 AS profiles
          ,0 AS conversations
          ,COUNT(*) AS events
          ,0 AS comments
      FROM flags
     WHERE site_id = $1
       AND item_type_id = 9
       AND item_is_deleted IS NOT TRUE
       AND item_is_moderated IS NOT TRUE
       AND parent_is_deleted IS NOT TRUE
       AND parent_is_moderated IS NOT TRUE
       AND microcosm_is_deleted IS NOT TRUE
       AND microcosm_is_moderated IS NOT TRUE
     UNION
    SELECT 0 AS profiles
          ,COUNT(*) AS conversations
          ,0 AS events
          ,0 AS comments
      FROM flags
     WHERE site_id = $1
       AND item_type_id = 6
       AND item_is_deleted IS NOT TRUE
       AND item_is_moderated IS NOT TRUE
       AND parent_is_deleted IS NOT TRUE
       AND parent_is_moderated IS NOT TRUE
       AND microcosm_is_deleted IS NOT TRUE
       AND microcosm_is_moderated IS NOT TRUE
     UNION
    SELECT 0 AS profiles
          ,0 AS conversations
          ,0 AS events
          ,COUNT(*) AS comments
      FROM flags
     WHERE site_id = $1
       AND item_type_id = 4
       AND parent_item_type_id <> 5
       AND item_is_deleted IS NOT TRUE
       AND item_is_moderated IS NOT TRUE
       AND parent_is_deleted IS NOT TRUE
       AND parent_is_moderated IS NOT TRUE
       AND microcosm_is_deleted IS NOT TRUE
       AND microcosm_is_moderated IS NOT TRUE
     UNION 
    SELECT COUNT(*) AS profiles
          ,0 AS conversations
          ,0 AS events
          ,0 AS comments
      FROM flags
     WHERE site_id = $1
       AND item_type_id = 3
       AND item_is_deleted IS NOT TRUE
       AND item_is_moderated IS NOT TRUE
       AND parent_is_deleted IS NOT TRUE
       AND parent_is_moderated IS NOT TRUE
       AND microcosm_is_deleted IS NOT TRUE
       AND microcosm_is_moderated IS NOT TRUE
     ) r`,
		siteId,
	).Scan(
		&stats.TotalProfiles,
		&stats.TotalConvs,
		&stats.TotalEvents,
		&stats.TotalComments,
	)

	return stats, err
}

func UpdateSiteStats(siteId int64) error {

	stats, err := CalcSiteStats(siteId)
	if err != nil {
		return err
	}

	db, err := h.GetConnection()
	if err != nil {
		return err
	}

	var exists bool
	err = db.QueryRow(`SELECT EXISTS (SELECT * from site_stats WHERE site_id = $1)`, siteId).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update
		_, err = db.Exec(
			`UPDATE site_stats SET 
               active_profiles = $2,
               online_profiles = $3,
               total_profiles = $4,
               total_conversations = $5,
               total_events = $6,
               total_comments = $7
            WHERE site_id = $1`,
			siteId,
			stats.ActiveProfiles,
			stats.OnlineProfiles,
			stats.TotalProfiles,
			stats.TotalConvs,
			stats.TotalEvents,
			stats.TotalComments,
		)
		if err != nil {
			return err
		}
	} else {
		// Insert
		_, err = db.Exec(
			`INSERT INTO site_stats (
               site_id,
               active_profiles,
               online_profiles,
               total_profiles,
               total_conversations,
               total_events,
               total_comments
            ) VALUES (
               $1, 
               $2,
               $3,
               $4,
               $5,
               $6,
               $7
            )`,
			siteId,
			stats.ActiveProfiles,
			stats.OnlineProfiles,
			stats.TotalProfiles,
			stats.TotalConvs,
			stats.TotalEvents,
			stats.TotalComments,
		)
		if err != nil {
			return err
		}
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteId)

	return nil
}

// Fetch and format statistics for a single site.
func GetSiteStats(siteId int64) ([]h.StatType, error) {

	// Try database.
	db, err := h.GetConnection()
	if err != nil {
		return []h.StatType{}, err
	}

	var stats SiteStatType
	err = db.QueryRow(
		`SELECT
           active_profiles,
           online_profiles,
           total_profiles,
           total_conversations,
           total_events,
           total_comments
         FROM
           site_stats
         WHERE
           site_id = $1`,
		siteId,
	).Scan(
		&stats.ActiveProfiles,
		&stats.OnlineProfiles,
		&stats.TotalProfiles,
		&stats.TotalConvs,
		&stats.TotalEvents,
		&stats.TotalComments,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Not in database, calculate synchronously. Should only
			// happen when site is newly created.
			stats, err = CalcSiteStats(siteId)
		} else {
			return []h.StatType{}, err
		}
	}

	jsonStats := []h.StatType{
		h.StatType{Metric: "activeProfiles", Value: stats.ActiveProfiles},
		h.StatType{Metric: "onlineProfiles", Value: stats.OnlineProfiles},
		h.StatType{Metric: "totalProfiles", Value: stats.TotalProfiles},
		h.StatType{Metric: "totalConversations", Value: stats.TotalConvs},
		h.StatType{Metric: "totalEvents", Value: stats.TotalEvents},
		h.StatType{Metric: "totalComments", Value: stats.TotalComments},
	}
	return jsonStats, nil
}

func GetSiteBySubdomain(subdomain string) (SiteType, int, error) {

	if strings.Trim(subdomain, " ") == "" {
		return SiteType{}, http.StatusBadRequest, errors.New(
			fmt.Sprintf("The domain key ('%s') cannot be empty", subdomain),
		)
	}

	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheSubdomain], subdomain)
	if val, ok := c.CacheGetInt64(mcKey); ok {
		return GetSite(val)
	}

	db, err := h.GetConnection()
	if err != nil {
		return SiteType{}, http.StatusInternalServerError, err
	}

	var siteId int64
	// The following query excludes is_deleted to prevent any the request
	// context from returning any site data, even though the site will still be
	// accessible via the control panel.
	err = db.QueryRow(`
SELECT site_id
  FROM sites
 WHERE subdomain_key = $1
   AND is_deleted IS NOT TRUE`,
		subdomain,
	).Scan(
		&siteId,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with subdomain %s not found", subdomain),
		)

	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	// Update cache
	c.CacheSetInt64(mcKey, siteId, mcTtl)

	return GetSite(siteId)
}

func GetSiteByDomain(domain string) (SiteType, int, error) {

	if strings.Trim(domain, " ") == "" {
		return SiteType{}, http.StatusBadRequest, errors.New(
			fmt.Sprintf("the supplied domain ('%s') cannot be empty.", domain),
		)
	}

	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheDomain], domain)
	if val, ok := c.CacheGetInt64(mcKey); ok {
		return GetSite(val)
	}

	db, err := h.GetConnection()
	if err != nil {
		return SiteType{}, http.StatusInternalServerError, err
	}

	var siteId int64
	// The following query excludes is_deleted to prevent any the request
	// context from returning any site data, even though the site will still be
	// accessible via the control panel.
	err = db.QueryRow(`
SELECT site_id
  FROM sites
 WHERE domain = $1
   AND is_deleted IS NOT TRUE`,
		domain,
	).Scan(
		&siteId,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound,
			errors.New(
				fmt.Sprintf("Resource with domain %s not found", domain),
			)
	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}

	// Update cache
	c.CacheSetInt64(mcKey, siteId, mcTtl)

	return GetSite(siteId)
}

func GetSites(
	userId int64,
	limit int64,
	offset int64,
) (
	[]SiteType,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return []SiteType{}, 0, 0, http.StatusInternalServerError, err
	}

	var sqlQuery string
	if userId > 0 {
		sqlQuery = `
SELECT COUNT(*) OVER() AS total
      ,s.site_id
  FROM sites s
  JOIN profiles p
    ON s.owned_by = p.profile_id
   AND s.site_id = p.site_id
 WHERE p.user_id = $1
   AND s.site_id <> 1
   AND s.is_deleted IS NOT TRUE
ORDER BY s.created ASC
 LIMIT $2
OFFSET $3`
	} else {
		sqlQuery = `
SELECT COUNT(*) OVER() AS total ,site_id
  FROM sites
 WHERE is_deleted IS NOT TRUE
   AND site_id <> 1
ORDER BY created ASC
 LIMIT $1
OFFSET $2`
	}

	var rows *sql.Rows
	if userId > 0 {
		rows, err = db.Query(sqlQuery, userId, limit, offset)
	} else {
		rows, err = db.Query(sqlQuery, limit, offset)
	}
	if err != nil {
		return []SiteType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Could not query rows: %v", err.Error()),
			)
	}
	defer rows.Close()

	var sites []SiteType
	var total int64

	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []SiteType{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}
		m, status, err := GetSite(id)
		if err != nil {
			return []SiteType{}, 0, 0, status, err
		}
		sites = append(sites, m)
	}
	err = rows.Err()
	if err != nil {
		return []SiteType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []SiteType{}, 0, 0, http.StatusBadRequest, errors.New(
			fmt.Sprintf("Offset (%d) would return an empty page.", offset),
		)
	}

	return sites, total, pages, http.StatusOK, nil
}

// CheckSiteHealth checks for valid domain, analytics, and logo/background
// settings.
func CheckSiteHealth(site SiteType) (SiteHealthType, int, error) {

	siteHealth := SiteHealthType{}
	if site.Id == 1 {
		return siteHealth, http.StatusBadRequest,
			errors.New("Cannot fetch status of root site.")
	}
	siteHealth.Site = site

	// If site has a custom domain set, check that a CNAME record with value
	// subdomain.microco.sm exists.
	if site.DomainNullable.Valid {
		siteHealth.DomainHealth.Set = true
		cname, err := net.LookupCNAME(site.Domain)
		if err != nil {
			siteHealth.DomainHealth.Valid = false
			siteHealth.DomainHealth.Error = err.Error()
		}
		siteURL := site.SubdomainKey + ".microco.sm"
		if cname != siteURL {
			siteHealth.DomainHealth.Valid = false
			siteHealth.DomainHealth.Error = fmt.Sprintf(
				"CNAME value is %s, expected %s",
				cname,
				siteURL,
			)
		}
		siteHealth.DomainHealth.Value = cname
	} else {
		siteHealth.DomainHealth.Set = false
	}

	// Check the site logo is reachable (if specified).
	if site.LogoUrl != "" {
		siteHealth.LogoUrlHealth.Set = true
		err := CheckSiteResource(site.LogoUrl, site)
		if err != nil {
			siteHealth.LogoUrlHealth.Valid = false
			siteHealth.LogoUrlHealth.Error = err.Error()
		} else {
			siteHealth.LogoUrlHealth.Valid = true
		}
		siteHealth.LogoUrlHealth.Value = site.LogoUrl
	} else {
		siteHealth.LogoUrlHealth.Set = false
	}

	// Check the header background URL is reachable (if specified).
	if site.BackgroundUrl != "" {
		siteHealth.BackgroundUrlHealth.Set = true
		err := CheckSiteResource(site.BackgroundUrl, site)
		if err != nil {
			siteHealth.BackgroundUrlHealth.Valid = false
			siteHealth.BackgroundUrlHealth.Error = err.Error()
		} else {
			siteHealth.BackgroundUrlHealth.Valid = true
		}
		siteHealth.BackgroundUrlHealth.Value = site.BackgroundUrl
	} else {
		siteHealth.BackgroundUrlHealth.Set = false
	}

	// Validate Google Analytics property (already done in validator, but
	// part of site health)
	if site.GaWebPropertyIdNullable.Valid {
		siteHealth.AnalyticsIDHealth.Set = true
		if strings.HasPrefix(site.GaWebPropertyId, "UA-") {
			siteHealth.AnalyticsIDHealth.Valid = true
		} else {
			siteHealth.AnalyticsIDHealth.Valid = false
			siteHealth.AnalyticsIDHealth.Error = fmt.Sprintf(
				"Invalid GA web property format: %s",
				site.GaWebPropertyId,
			)
		}
		// We've already checked validity so ignore error.
		value, _ := site.GaWebPropertyIdNullable.Value()
		siteHealth.AnalyticsIDHealth.Value = value
	} else {
		siteHealth.AnalyticsIDHealth.Set = false
	}
	return siteHealth, http.StatusOK, nil
}

// CheckSiteResource is a utility for formatting site attribute URLs and
// determining whether they are reachable.
func CheckSiteResource(resource string, site SiteType) error {
	resourceURL, err := url.Parse(resource)
	if err != nil {
		return err
	}

	// If URL is not absolute, prepend site domain (if specified) or Microcosm
	// domain.
	if !resourceURL.IsAbs() {
		resourceURL.Scheme = "http"
		if site.DomainNullable.Valid {
			resourceURL.Host = site.Domain
		} else {
			resourceURL.Host = site.SubdomainKey + ".microco.sm"
		}
	}

	// Retrieve the logo and handle any HTTP protocol errors or a non-successful
	// HTTP status code.
	resp, err := http.Get(resourceURL.String())
	if err != nil {
		glog.Warningf(
			"Transport error retrieving logo at %s: %s",
			resourceURL.String(),
			err.Error(),
		)
		return err
	}
	if resp.StatusCode >= 400 {
		return err
	}
	return nil
}
