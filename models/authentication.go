package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

type AccessTokenType struct {
	AccessTokenId int64     `json:"-"`
	TokenValue    string    `json:"accessToken"`
	UserId        int64     `json:"-"`
	User          UserType  `json:"user"`
	ClientId      int64     `json:"clientId"`
	Created       time.Time `json:"created"`
	Expires       time.Time `json:"expires"`
}

type OauthClientType struct {
	ClientId     int64
	Name         string
	Created      time.Time
	ClientSecret string
}

type AccessTokenRequestType struct {
	Assertion    string
	ClientSecret string
}

type PersonaRequestType struct {
	Assertion string `json:"assertion"`
	Audience  string `json:"audience"`
}

type PersonaResponseType struct {
	Status   string
	Email    string
	Audience string
	Expires  int32
	Issuer   string
}

func (m *AccessTokenType) Insert() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	err = tx.QueryRow(`
INSERT INTO access_tokens (
    token_value, user_id, client_id
) VALUES (
    $1, $2, $3
) RETURNING access_token_id, created, expires`,
		m.TokenValue,
		m.UserId,
		m.ClientId,
	).Scan(
		&m.AccessTokenId,
		&m.Created,
		&m.Expires,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error inserting data and returning ID: %+v", err),
			)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	// Put access token into memcache
	if m.UserId > 0 {
		u, status, err := GetUser(m.UserId)
		if err != nil {
			return status, err
		}
		m.User = u
	}

	// Update cache
	mcKey := fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], m.TokenValue)
	c.CacheSet(mcKey, m, int32(m.Expires.Sub(time.Now()).Seconds()))

	return http.StatusOK, nil
}

func GetAccessToken(token string) (AccessTokenType, int, error) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], token)
	if val, ok := c.CacheGet(mcKey, AccessTokenType{}); ok {
		return val.(AccessTokenType), http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return AccessTokenType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Connection failed: %v", err.Error()),
		)
	}

	var m AccessTokenType

	err = db.QueryRow(`
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
		&m.AccessTokenId,
		&m.TokenValue,
		&m.UserId,
		&m.ClientId,
		&m.Created,
		&m.Expires,
	)
	if err == sql.ErrNoRows {
		return AccessTokenType{}, http.StatusNotFound,
			errors.New("Token not found")

	} else if err != nil {
		return AccessTokenType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.UserId > 0 {
		u, status, err := GetUser(m.UserId)
		if err != nil {
			return AccessTokenType{}, status, err
		}
		m.User = u
	}

	// Update cache
	c.CacheSet(mcKey, m, int32(m.Expires.Sub(time.Now()).Seconds()))

	return m, http.StatusOK, nil
}

func (m *AccessTokenType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM access_tokens 
 WHERE token_value = $1`,
		m.TokenValue,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not delete token: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not commit transaction: %v", err.Error()),
		)
	}

	// Clear the cache. We do this manually as the ID in this case isn't
	// an int64
	c.CacheDelete(fmt.Sprintf(mcAccessTokenKeys[c.CacheDetail], m.TokenValue))

	return http.StatusOK, nil
}

func RetrieveClientBySecret(secret string) (OauthClientType, error) {

	db, err := h.GetConnection()
	if err != nil {
		return OauthClientType{}, err
	}

	rows, err := db.Query(`
SELECT client_id
      ,name
      ,created
      ,client_secret
  FROM oauth_clients
 WHERE client_secret = $1`,
		secret,
	)
	defer rows.Close()

	var m OauthClientType

	for rows.Next() {
		m = OauthClientType{}
		err = rows.Scan(
			&m.ClientId,
			&m.Name,
			&m.Created,
			&m.ClientSecret,
		)
		if err != nil {
			return OauthClientType{}, errors.New(fmt.Sprintf(
				"Row parsing error: %v", err.Error()),
			)
		}
	}
	err = rows.Err()
	if err != nil {
		return OauthClientType{}, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	if m.ClientId == 0 {
		return OauthClientType{}, errors.New("Invalid client secret")
	}

	return m, nil
}
