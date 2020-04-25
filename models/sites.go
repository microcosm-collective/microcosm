package models

import (
	"database/sql"
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
	// DefaultThemeID is hard-coded and matches the theme in the microweb
	// docroot
	DefaultThemeID int64 = 1

	// DefaultBackgroundColor in hex
	DefaultBackgroundColor string = `#FFFFFF`

	// DefaultBackgroundPosition for any background CSS image
	DefaultBackgroundPosition string = `tile`

	// DefaultLinkColor in hex
	DefaultLinkColor string = `#4082C3`
)

// DisallowedSubdomains is a list of all subdomains that cannot be registered
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
	"policy",        // Legal
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

// SitesType is an array of sites
type SitesType struct {
	Sites h.ArrayType    `json:"sites"`
	Meta  h.CoreMetaType `json:"meta"`
}

// SiteType is the grandaddy of all types and describes a site
type SiteType struct {
	ID                        int64          `json:"siteId"`
	SiteURL                   string         `json:"siteURL"`
	Title                     string         `json:"title"`
	Description               string         `json:"description"`
	SubdomainKey              string         `json:"subdomainKey"`
	Domain                    string         `json:"domain"`
	DomainNullable            sql.NullString `json:"-"`
	ForceSSL                  bool           `json:"forceSSL,omitempty"`
	OwnedByID                 int64          `json:"-"`
	OwnedBy                   interface{}    `json:"ownedBy"`
	ThemeID                   int64          `json:"themeId"`
	LogoURL                   string         `json:"logoUrl"`
	FaviconURL                string         `json:"faviconUrl,omitempty"`
	BackgroundColor           string         `json:"backgroundColor"`
	BackgroundURL             string         `json:"backgroundUrl,omitempty"`
	BackgroundPosition        string         `json:"backgroundPosition,omitempty"`
	LinkColor                 string         `json:"linkColor"`
	GaWebPropertyID           string         `json:"gaWebPropertyId,omitempty"`
	GaWebPropertyIDNullable   sql.NullString `json:"-"`
	Menu                      []h.LinkType   `json:"menu"`
	Auth0DomainNullable       sql.NullString `json:"-"`
	Auth0Domain               string         `json:"auth0Domain,omitempty"`
	Auth0ClientIDNullable     sql.NullString `json:"-"`
	Auth0ClientID             string         `json:"auth0ClientId,omitempty"`
	Auth0ClientSecretNullable sql.NullString `json:"-"`
	Auth0ClientSecret         string         `json:"-"`

	Meta struct {
		h.CreatedType
		h.EditedType

		Flags struct {
			Deleted bool `json:"deleted"`
		} `json:"flags,omitempty"`

		h.CoreMetaType
	} `json:"meta"`
}

// SiteStatType encapsulates global stats for the site
type SiteStatType struct {
	ActiveProfiles int64
	OnlineProfiles int64
	TotalProfiles  int64
	TotalConvs     int64
	TotalEvents    int64
	TotalComments  int64
}

// SiteHealthType encapsulates the state of the site configuration
type SiteHealthType struct {
	Site                SiteType            `json:"site"`
	DomainHealth        SiteHealthAttribute `json:"domainHealth"`
	BackgroundURLHealth SiteHealthAttribute `json:"backgroundUrlHealth"`
	LogoURLHealth       SiteHealthAttribute `json:"logoUrlHealth"`
	AnalyticsIDHealth   SiteHealthAttribute `json:"analyticsIDHealth"`
}

// SiteHealthAttribute encapsulates a state for site configuration
type SiteHealthAttribute struct {
	Set   bool        `json:"set"`
	Valid bool        `json:"valid"`
	Error string      `json:"error"`
	Value interface{} `json:"value"`
}

var regAlphaNum = regexp.MustCompile(`[A-Za-z0-9]+`)

// Validate returns true if the site data is good
func (m *SiteType) Validate(exists bool) (int, error) {
	preventShouting := false
	m.Title = CleanSentence(m.Title, preventShouting)
	m.Description = CleanBlockText(m.Description)
	m.SubdomainKey = CleanWord(m.SubdomainKey)
	m.Domain = CleanWord(m.Domain)

	if exists {
		if m.ID < 1 {
			return http.StatusBadRequest, fmt.Errorf("Invalid site ID")
		}
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("You must specify a site title")
	}

	if strings.Trim(m.Description, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("You must specify a site description")
	}

	if strings.Trim(m.SubdomainKey, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("You must specify a subdomain key")
	}

	if m.SubdomainKey != "" {
		for _, subdomain := range DisallowedSubdomains {
			if m.SubdomainKey == subdomain {
				return http.StatusBadRequest,
					fmt.Errorf(
						"Subdomain '%s' is reserved and cannot be used",
						m.SubdomainKey,
					)
			}
		}
		if !regAlphaNum.MatchString(m.SubdomainKey) {
			return http.StatusBadRequest,
				fmt.Errorf("Subdomain key must be alphanumeric")
		}
	}

	m.BackgroundColor = strings.Trim(m.BackgroundColor, " ")
	if m.BackgroundColor == "" {
		m.BackgroundColor = DefaultBackgroundColor
	}

	if !h.IsValidColor(m.BackgroundColor) {
		return http.StatusBadRequest,
			fmt.Errorf(
				"Background color is not a valid HTML color (hex or named)",
			)
	}

	m.LinkColor = strings.Trim(m.LinkColor, " ")
	if m.LinkColor == "" {
		m.LinkColor = DefaultLinkColor
	}

	if !h.IsValidColor(m.LinkColor) {
		return http.StatusBadRequest,
			fmt.Errorf("Link color is not a valid HTML color (hex or named)")
	}

	validBackgroundPosition := map[string]bool{
		"cover":  true,
		"tall":   true,
		"wide":   true,
		"left":   true,
		"center": true,
		"right":  true,
		"tiled":  true,
	}

	m.BackgroundPosition = strings.Trim(strings.ToLower(m.BackgroundPosition), " ")
	if !validBackgroundPosition[m.BackgroundPosition] {
		m.BackgroundPosition = DefaultBackgroundPosition
	}

	if m.GaWebPropertyID != "" {
		if !strings.HasPrefix(m.GaWebPropertyID, "UA-") {
			return http.StatusBadRequest,
				fmt.Errorf(
					"gaWebPropertyId must be in the form of the UA-XXXX-Y " +
						"property ID that Google Analytics provided to you",
				)
		}
		m.GaWebPropertyIDNullable = sql.NullString{
			String: m.GaWebPropertyID,
			Valid:  true,
		}
	}

	return http.StatusOK, nil
}

// Hydrate populates a partially populated site
func (m *SiteType) Hydrate() (int, error) {

	profile, status, err := GetProfileSummary(m.ID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	profile, status, err = GetProfileSummary(m.ID, m.OwnedByID)
	if err != nil {
		return status, err
	}
	m.OwnedBy = profile

	// Site stats are also pulled from cache so that the site itself can be
	// cached for a long time, but the stats can be updated more frequently.
	// These are updated by eviction from a cron job, so have a long TTL in case
	// the cron job fails
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheCounts], m.ID)
	if val, ok := c.Get(mcKey, []h.StatType{}); ok {
		m.Meta.Stats = val.([]h.StatType)
	} else {
		stats, err := GetSiteStats(m.ID)
		if err != nil {
			glog.Error(err)
		} else {
			m.Meta.Stats = stats
			c.Set(mcKey, m.Meta.Stats, mcTTL)
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
			fmt.Errorf("Error fetching site by subdomain: %+v", err)
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
	profile.UserID = user.ID
	profile.Visible = true

	tx, err := h.GetTransaction()
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
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
		site.ThemeID,
		user.ID,
		profile.ProfileName,

		profile.AvatarIDNullable,
		profile.AvatarURLNullable,
		site.DomainNullable,
		site.Description,
		site.LogoURL,

		site.BackgroundURL,
		site.BackgroundPosition,
		site.BackgroundColor,
		site.LinkColor,
	)
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Error executing query: %v", err.Error())
	}
	defer rows.Close()

	var siteID int64
	var profileID int64
	for rows.Next() {
		err = rows.Scan(&siteID, &profileID)
		if err != nil {
			return SiteType{}, ProfileType{}, http.StatusInternalServerError,
				fmt.Errorf("Error inserting data and returning IDs: %v", err.Error())
		}
	}
	if rows.Err() != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning IDs: %v", rows.Err().Error())
	}
	rows.Close()

	site.ID = siteID
	profile.SiteID = siteID
	profile.ID = profileID

	// Create profile_options record for the newly created profile
	profileOptions, _, err := GetProfileOptionsDefaults(site.ID)
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not load default profile options: %v", err.Error())
	}
	profileOptions.ProfileID = profile.ID

	status, err = profileOptions.Insert(tx)
	if err != nil {
		return SiteType{}, ProfileType{}, status,
			fmt.Errorf("Could not insert new profile options: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	// Create attachment for avatar and attach it to profile
	fm, _, err := StoreGravatar(MakeGravatarURL(profile.ProfileName))
	if err != nil {
		return SiteType{}, ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not store gravatar for profile: %+v", err)
	}

	// Attach avatar to profile
	attachment, status, err := AttachAvatar(profile.ID, fm)
	if err != nil {
		return SiteType{}, ProfileType{}, status,
			fmt.Errorf("Could not attach avatar to profile: %v", err.Error())
	}

	// Construct URL to avatar, update profile with Avatar ID and URL
	filePath := fm.FileHash
	if fm.FileExt != "" {
		filePath += `.` + fm.FileExt
	}
	profile.AvatarURLNullable = sql.NullString{
		String: fmt.Sprintf("%s/%s", h.APITypeFile, filePath),
		Valid:  true,
	}
	profile.AvatarIDNullable = sql.NullInt64{
		Int64: attachment.AttachmentID,
		Valid: true,
	}
	status, err = profile.Update()
	if err != nil {
		return SiteType{}, ProfileType{}, status,
			fmt.Errorf("Could not update profile with avatar: %v", err.Error())
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
	email.Send(site.ID)

	return site, profile, http.StatusOK, nil
}

// Update updates a site
func (m *SiteType) Update() (int, error) {

	status, err := m.Validate(true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
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
		m.ID,

		m.Title,
		m.Description,
		m.DomainNullable,
		m.ThemeID,
		m.LogoURL,
		m.FaviconURL,

		m.BackgroundURL,
		m.BackgroundColor,
		m.BackgroundPosition,
		m.LinkColor,
		m.GaWebPropertyIDNullable,

		m.Meta.Flags.Deleted,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("Update of site failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeSite], m.ID)

	return http.StatusOK, nil
}

// Delete will remove a site from the database
func (m *SiteType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM sites
 WHERE site_id = $1`,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Delete failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeSite], m.ID)

	return http.StatusOK, nil
}

// GetURL builds a site URL depending on whether or not it is a subdomain or
// custom domain
func (m *SiteType) GetURL() string {
	if m.Domain == "" {
		return "https://" + m.SubdomainKey + ".microco.sm"
	}

	if m.ForceSSL {
		return "https://" + m.Domain
	}

	return "http://" + m.Domain
}

// GetSiteTitle returns (cheaply) the site title
func GetSiteTitle(id int64) string {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheTitle], id)
	if val, ok := c.GetString(mcKey); ok {
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
	c.SetString(mcKey, title, mcTTL)

	return title
}

// GetSite returns a site
func GetSite(id int64) (SiteType, int, error) {
	// Try cache
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, SiteType{}); ok {
		m := val.(SiteType)
		// Site now caches the profile summaries as the benefits on performance
		// are too great to do this every time as context.MakeContext for every
		// controller uses a Site object
		// m.Hydrate()
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
      ,force_ssl
      ,auth0_domain
      ,auth0_client_id
      ,auth0_client_secret
  FROM sites s
      ,themes t
 WHERE s.theme_id = t.theme_id
   AND s.site_id = $1`,
		id,
	).Scan(
		&m.ID,
		&m.Title,
		&m.Description,
		&m.SubdomainKey,
		&m.DomainNullable,
		&m.Meta.Created,
		&m.Meta.CreatedByID,
		&m.OwnedByID,
		&m.ThemeID,
		&m.LogoURL,
		&m.BackgroundURL,
		&m.FaviconURL,
		&m.BackgroundColor,
		&m.BackgroundPosition,
		&m.LinkColor,
		&m.GaWebPropertyIDNullable,
		&m.Meta.Flags.Deleted,
		&m.ForceSSL,
		&m.Auth0DomainNullable,
		&m.Auth0ClientIDNullable,
		&m.Auth0ClientSecretNullable,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound,
			fmt.Errorf("Resource with site ID %d not found", id)
	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	if m.DomainNullable.Valid {
		m.Domain = m.DomainNullable.String
	}
	// Set the definitive siteURL
	m.SiteURL = m.GetURL()

	if m.GaWebPropertyIDNullable.Valid {
		m.GaWebPropertyID = m.GaWebPropertyIDNullable.String
	}
	if m.Auth0DomainNullable.Valid {
		m.Auth0Domain = m.Auth0DomainNullable.String
	}
	if m.Auth0ClientIDNullable.Valid {
		m.Auth0ClientID = m.Auth0ClientIDNullable.String
	}
	if m.Auth0ClientSecretNullable.Valid {
		m.Auth0ClientSecret = m.Auth0ClientSecretNullable.String
	}
	menu, status, err := GetMenu(m.ID)
	if err != nil {
		return SiteType{}, status,
			fmt.Errorf("Error fetching menu: %v", err.Error())
	}
	m.Menu = menu
	if m.BackgroundURL == "" {
		m.BackgroundPosition = ""
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeSite, m.ID),
			h.GetLink("microcosm", "", h.ItemTypeMicrocosm, 0),
			h.GetLink("profile", "", h.ItemTypeProfile, 0),
			h.LinkType{Rel: "legal", Href: "/api/v1/legal"},
		}
	m.Hydrate()

	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// CalcSiteStats is expensive and should not be run to synchronously service a
// request.
func CalcSiteStats(siteID int64) (SiteStatType, error) {
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
		siteID,
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
		siteID,
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
		siteID,
	).Scan(
		&stats.TotalProfiles,
		&stats.TotalConvs,
		&stats.TotalEvents,
		&stats.TotalComments,
	)

	return stats, err
}

// UpdateSiteStats updates the stats for a given site
func UpdateSiteStats(siteID int64) error {
	stats, err := CalcSiteStats(siteID)
	if err != nil {
		return err
	}

	db, err := h.GetConnection()
	if err != nil {
		return err
	}

	var exists bool
	err = db.QueryRow(
		`SELECT EXISTS (SELECT * from site_stats WHERE site_id = $1)`,
		siteID,
	).Scan(&exists)
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
			siteID,
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
			siteID,
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

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteID)

	return nil
}

// GetSiteStats fetches and formats the statistics for a single site.
func GetSiteStats(siteID int64) ([]h.StatType, error) {

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
		siteID,
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
			stats, err = CalcSiteStats(siteID)
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

// GetSiteBySubdomain returns a site for a given subdomain key
func GetSiteBySubdomain(subdomain string) (SiteType, int, error) {
	if subdomain == "gfora" {
		subdomain = "lfgss"
	}

	if strings.Trim(subdomain, " ") == "" {
		return SiteType{}, http.StatusBadRequest,
			fmt.Errorf("The domain key ('%s') cannot be empty", subdomain)
	}

	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheSubdomain], subdomain)
	if val, ok := c.GetInt64(mcKey); ok {
		return GetSite(val)
	}

	db, err := h.GetConnection()
	if err != nil {
		return SiteType{}, http.StatusInternalServerError, err
	}

	var siteID int64
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
		&siteID,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound,
			fmt.Errorf("Resource with subdomain %s not found", subdomain)

	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	// Update cache
	c.SetInt64(mcKey, siteID, mcTTL)

	return GetSite(siteID)
}

// GetSiteByDomain returns a site for a given custom domain name
func GetSiteByDomain(domain string) (SiteType, int, error) {

	if domain == "www.gfora.com" {
		domain = "www.lfgss.com"
	}

	if strings.Trim(domain, " ") == "" {
		return SiteType{}, http.StatusBadRequest,
			fmt.Errorf("the supplied domain ('%s') cannot be empty", domain)
	}

	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheDomain], domain)
	if val, ok := c.GetInt64(mcKey); ok {
		return GetSite(val)
	}

	db, err := h.GetConnection()
	if err != nil {
		return SiteType{}, http.StatusInternalServerError, err
	}

	var siteID int64
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
		&siteID,
	)
	if err == sql.ErrNoRows {
		return SiteType{}, http.StatusNotFound,
			fmt.Errorf("Resource with domain %s not found", domain)
	} else if err != nil {
		return SiteType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	// Update cache
	c.SetInt64(mcKey, siteID, mcTTL)

	return GetSite(siteID)
}

// GetSites returns a list of sites owned by a given user
func GetSites(
	userID int64,
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
	if userID > 0 {
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
	if userID > 0 {
		rows, err = db.Query(sqlQuery, userID, limit, offset)
	} else {
		rows, err = db.Query(sqlQuery, limit, offset)
	}
	if err != nil {
		return []SiteType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Could not query rows: %v", err.Error())
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
				fmt.Errorf("Row parsing error: %v", err.Error())
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
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []SiteType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("Offset (%d) would return an empty page", offset)
	}

	return sites, total, pages, http.StatusOK, nil
}

// CheckSiteHealth checks for valid domain, analytics, and logo/background
// settings.
func CheckSiteHealth(site SiteType) (SiteHealthType, int, error) {

	siteHealth := SiteHealthType{}
	if site.ID == 1 {
		return siteHealth, http.StatusBadRequest,
			fmt.Errorf("Cannot fetch status of root site")
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
	if site.LogoURL != "" {
		siteHealth.LogoURLHealth.Set = true
		err := CheckSiteResource(site.LogoURL, site)
		if err != nil {
			siteHealth.LogoURLHealth.Valid = false
			siteHealth.LogoURLHealth.Error = err.Error()
		} else {
			siteHealth.LogoURLHealth.Valid = true
		}
		siteHealth.LogoURLHealth.Value = site.LogoURL
	} else {
		siteHealth.LogoURLHealth.Set = false
	}

	// Check the header background URL is reachable (if specified).
	if site.BackgroundURL != "" {
		siteHealth.BackgroundURLHealth.Set = true
		err := CheckSiteResource(site.BackgroundURL, site)
		if err != nil {
			siteHealth.BackgroundURLHealth.Valid = false
			siteHealth.BackgroundURLHealth.Error = err.Error()
		} else {
			siteHealth.BackgroundURLHealth.Valid = true
		}
		siteHealth.BackgroundURLHealth.Value = site.BackgroundURL
	} else {
		siteHealth.BackgroundURLHealth.Set = false
	}

	// Validate Google Analytics property (already done in validator, but
	// part of site health)
	if site.GaWebPropertyIDNullable.Valid {
		siteHealth.AnalyticsIDHealth.Set = true
		if strings.HasPrefix(site.GaWebPropertyID, "UA-") {
			siteHealth.AnalyticsIDHealth.Valid = true
		} else {
			siteHealth.AnalyticsIDHealth.Valid = false
			siteHealth.AnalyticsIDHealth.Error = fmt.Sprintf(
				"Invalid GA web property format: %s",
				site.GaWebPropertyID,
			)
		}
		// We've already checked validity so ignore error.
		value, _ := site.GaWebPropertyIDNullable.Value()
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
