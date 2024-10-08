package jwt

import (
	"errors"
	"log"
	"time"

	jwtGo "github.com/golang-jwt/jwt/v5"
)

type jwtToken struct {
	Token    *jwtGo.Token
	ParseErr error
	options  tokenOptions
}

type tokenOptions struct {
	ValidTime           time.Duration
	SigningMethodString string
	Debug               bool
}

func (t *jwtToken) myLog(stoofs interface{}) {
	if t.options.Debug {
		log.Println(stoofs)
	}
}

func (c *credentials) buildTokenWithClaimsFromString(tokenString string, verifyKey interface{}, validTime time.Duration) *jwtToken {
	// note @adam-hanna: should we be checking inputs? Especially the token string?
	var newToken jwtToken

	token, err := jwtGo.ParseWithClaims(tokenString, &ClaimsType{}, func(token *jwtGo.Token) (interface{}, error) {
		if token.Method != jwtGo.GetSigningMethod(c.options.SigningMethodString) {
			c.myLog("Incorrect singing method on token")
			return nil, errors.New("incorrect singing method on token")
		}
		return verifyKey, nil
	})

	if token == nil {
		token = new(jwtGo.Token)
		token.Claims = new(ClaimsType)
		c.myLog("token is nil, set empty token (parse error=" + err.Error() + ")")
	}

	newToken.Token = token
	newToken.ParseErr = err

	newToken.options.ValidTime = validTime
	newToken.options.SigningMethodString = c.options.SigningMethodString
	newToken.options.Debug = c.options.Debug

	return &newToken
}

func (c *credentials) newTokenWithClaims(claims *ClaimsType, validTime time.Duration) *jwtToken {
	var newToken jwtToken

	newToken.Token = jwtGo.NewWithClaims(jwtGo.GetSigningMethod(c.options.SigningMethodString), claims)
	newToken.ParseErr = nil
	newToken.options.ValidTime = validTime
	newToken.options.SigningMethodString = c.options.SigningMethodString
	newToken.options.Debug = c.options.Debug

	return &newToken
}

func (t *jwtToken) updateTokenExpiry() *jwtError {
	tokenClaims, ok := t.Token.Claims.(*ClaimsType)
	if !ok {
		return newJwtError(errors.New("cannot read token claims"), 500)
	}

	tokenClaims.RegisteredClaims.ExpiresAt = jwtGo.NewNumericDate(time.Now().Add(t.options.ValidTime))
	// tokenClaims.RegisteredClaims.ExpiresAt = jwtGo.NewNumericDate(time.Now().Add(t.options.ValidTime).Unix())

	// update the token
	t.Token = jwtGo.NewWithClaims(jwtGo.GetSigningMethod(t.options.SigningMethodString), tokenClaims)

	return nil
}

func (t *jwtToken) updateTokenCsrf(csrfString string) *jwtError {
	tokenClaims, ok := t.Token.Claims.(*ClaimsType)
	if !ok {
		return newJwtError(errors.New("cannot read token claims"), 500)
	}

	tokenClaims.Csrf = csrfString

	// update the token
	t.Token = jwtGo.NewWithClaims(jwtGo.GetSigningMethod(t.options.SigningMethodString), tokenClaims)

	return nil
}

func (t *jwtToken) updateTokenExpiryAndCsrf(csrfString string) *jwtError {
	tokenClaims, ok := t.Token.Claims.(*ClaimsType)
	if !ok {
		return newJwtError(errors.New("cannot read token claims"), 500)
	}

	tokenClaims.RegisteredClaims.ExpiresAt = jwtGo.NewNumericDate(time.Now().Add(t.options.ValidTime))
	tokenClaims.Csrf = csrfString

	// update the token
	t.Token = jwtGo.NewWithClaims(jwtGo.GetSigningMethod(t.options.SigningMethodString), tokenClaims)

	return nil
}
