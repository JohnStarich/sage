package server

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/redactor"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const (
	tokenLength          = 64
	tokenDuration        = 5 * time.Minute
	refreshTokenLength   = 128
	refreshTokenDuration = 8 * time.Hour

	authHeaderName      = "Authorization"
	setCookieHeaderName = "Set-Cookie"
	tokenCookieName     = "token"
)

var (
	errInvalidLogin = errors.New("Incorrect password")
	errUnauthorized = errors.New("Unauthorized")
)

type authenticator struct {
	password              redactor.String
	tokens, refreshTokens *cache.Cache
}

func newAuthenticator(password redactor.String) *authenticator {
	return &authenticator{
		password:      password,
		tokens:        cache.New(tokenDuration, tokenDuration/5+1),
		refreshTokens: cache.New(refreshTokenDuration, refreshTokenDuration/5+1),
	}
}

func (a *authenticator) SignIn(password redactor.String) (token, refreshToken string, tokenExpire, refreshExpire time.Time, err error) {
	if a.password != password {
		return "", "", time.Time{}, time.Time{}, errInvalidLogin
	}
	now := time.Now()
	token = randomToken(tokenLength)
	a.tokens.SetDefault(token, true)
	refreshToken = randomToken(refreshTokenLength)
	a.refreshTokens.SetDefault(refreshToken, true)
	return token, refreshToken, now.Add(tokenDuration), now.Add(refreshTokenDuration), nil
}

func (a *authenticator) Authenticate(resp http.ResponseWriter, req *http.Request) error {
	authTokenHeader := req.Header.Get(authHeaderName)
	if authTokenHeader != "" {
		// authentication provided via header
		if _, found := a.tokens.Get(authTokenHeader); found {
			// valid token
			return nil
		}
		token, expires, err := a.NewToken(authTokenHeader)
		if err != nil {
			// auth header was not a valid refresh token
			return err
		}
		// successfully created new token
		a.SetCookies(resp, token, expires)
		return nil
	}
	// authentication provided via cookie
	tokenCookie, err := req.Cookie(tokenCookieName)
	if err != nil {
		return errUnauthorized
	}
	if _, found := a.tokens.Get(tokenCookie.Value); !found {
		return errUnauthorized
	}
	return nil
}

func (a *authenticator) NewToken(refreshToken string) (string, time.Time, error) {
	_, found := a.refreshTokens.Get(refreshToken)
	if !found || refreshToken == "" {
		// !found == expired or doesn't exist
		return "", time.Time{}, errUnauthorized
	}
	token := randomToken(tokenLength)
	expiration := time.Now().Add(tokenDuration)
	a.tokens.SetDefault(token, true)
	return token, expiration, nil
}

func (a *authenticator) SetCookies(resp http.ResponseWriter, token string, expireTime time.Time) {
	resp.Header().Add(setCookieHeaderName, (&http.Cookie{
		Name:    tokenCookieName,
		Value:   token,
		Expires: expireTime.Add(-10 * time.Second), // expire a little earlier on client to account for latency
	}).String())
}

func requireAuth(auth *authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := auth.Authenticate(c.Writer, c.Request)
		if err == nil {
			return
		}

		if err == errUnauthorized {
			abortWithClientError(c, http.StatusUnauthorized, err)
			return
		}
		abortWithClientError(c, http.StatusInternalServerError, err)
	}
}

func signIn(auth *authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var creds struct {
			Password redactor.String
		}
		if err := c.BindJSON(&creds); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		token, refreshToken, tokenExpires, refreshTokenExpires, err := auth.SignIn(creds.Password)
		if err != nil {
			abortWithClientError(c, http.StatusUnauthorized, err)
			return
		}
		auth.SetCookies(c.Writer, token, tokenExpires)
		c.JSON(http.StatusOK, map[string]interface{}{
			"Token":               token,
			"TokenExpires":        tokenExpires,
			"RefreshToken":        refreshToken,
			"RefreshTokenExpires": refreshTokenExpires,
		})
	}
}

func randomToken(length uint) string {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	if err != nil {
		panic("Error generating random string")
	}
	return base64.StdEncoding.EncodeToString(buf)
}
