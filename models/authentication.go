package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// AccessTokenType describes an access token
type AccessTokenType struct {
	AccessTokenID int64     `json:"-"`
	TokenValue    string    `json:"accessToken"`
	UserID        int64     `json:"-"`
	User          UserType  `json:"user"`
	ClientID      int64     `json:"clientId"`
	Created       time.Time `json:"created"`
	Expires       time.Time `json:"expires"`
}

// OAuthClientType describes an OAuth client
type OAuthClientType struct {
	ClientID     int64
	Name         string
	Created      time.Time
	ClientSecret string
}

// AccessTokenRequestType is a request for an access token
type AccessTokenRequestType struct {
	Assertion    string
	ClientSecret string
}

// PersonaRequestType is a Mozilla Persona request
type PersonaRequestType struct {
	Assertion string `json:"assertion"`
	Audience  string `json:"audience"`
}

// PersonaResponseType is a Mozilla Persona response
type PersonaResponseType struct {
	Status   string
	Email    string
	Audience string
	Expires  int32
	Issuer   string
}

// Insert saves an access token
func (m *AccessTokenType) Insert() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	err = tx.QueryRow(`--InsertAccessToken
INSERT INTO access_tokens (
    token_value, user_id, client_id
) VALUES (
    $1, $2, $3
) RETURNING access_token_id, created, expires`,
		m.TokenValue,
		m.UserID,
		m.ClientID,
	).Scan(
		&m.AccessTokenID,
		&m.Created,
		&m.Expires,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error inserting data and returning ID: %+v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	// Put access token into memcache
	if m.UserID > 0 {
		u, status, err := GetUser(m.UserID)
		if err != nil {
			return status, err
		}
		m.User = u
	}

	// Update cache
	mcKey := fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], m.TokenValue)

	c.Set(mcKey, m, int32(time.Until(m.Expires).Seconds()))

	return http.StatusOK, nil
}

// GetAccessToken returns an access token
func GetAccessToken(token string) (AccessTokenType, int, error) {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], token)
	if val, ok := c.Get(mcKey, AccessTokenType{}); ok {
		return val.(AccessTokenType), http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return AccessTokenType{}, http.StatusInternalServerError,
			fmt.Errorf("connection failed: %v", err.Error())
	}

	var m AccessTokenType

	err = db.QueryRow(`--GetAccessToken
SELECT access_token_id
      ,token_value
      ,user_id
      ,client_id
      ,created
      ,expires
  FROM access_tokens
 WHERE token_value = $1`,
		token,
	).Scan(
		&m.AccessTokenID,
		&m.TokenValue,
		&m.UserID,
		&m.ClientID,
		&m.Created,
		&m.Expires,
	)
	if err == sql.ErrNoRows {
		return AccessTokenType{}, http.StatusNotFound,
			fmt.Errorf("token not found")

	} else if err != nil {
		return AccessTokenType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	if m.UserID > 0 {
		u, status, err := GetUser(m.UserID)
		if err != nil {
			return AccessTokenType{}, status, err
		}
		m.User = u
	}

	// Update cache
	c.Set(mcKey, m, int32(time.Until(m.Expires).Seconds()))

	return m, http.StatusOK, nil
}

// Delete removes an access token
func (m *AccessTokenType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--DeleteAccessToken
DELETE FROM access_tokens 
 WHERE token_value = $1`,
		m.TokenValue,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not delete token: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not commit transaction: %v", err.Error())
	}

	// Clear the cache. We do this manually as the ID in this case isn't
	// an int64
	c.Delete(fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], m.TokenValue))

	return http.StatusOK, nil
}

// RetrieveClientBySecret fetches a client from a client secret
func RetrieveClientBySecret(secret string) (OAuthClientType, error) {
	db, err := h.GetConnection()
	if err != nil {
		return OAuthClientType{}, err
	}

	rows, _ := db.Query(`--GetOAuthClientBySecret
SELECT client_id
      ,name
      ,created
      ,client_secret
  FROM oauth_clients
 WHERE client_secret = $1`,
		secret,
	)
	defer rows.Close()

	var m OAuthClientType

	for rows.Next() {
		m = OAuthClientType{}
		err = rows.Scan(
			&m.ClientID,
			&m.Name,
			&m.Created,
			&m.ClientSecret,
		)
		if err != nil {
			return OAuthClientType{},
				fmt.Errorf("row parsing error: %v", err.Error())
		}
	}
	err = rows.Err()
	if err != nil {
		return OAuthClientType{},
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	if m.ClientID == 0 {
		return OAuthClientType{}, fmt.Errorf("invalid client secret")
	}

	return m, nil
}
