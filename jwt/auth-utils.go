package jwt

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// return is (authTokenString, refreshTokenString, err)
func (a *Auth) extractTokenStringsFromReq(r *http.Request) (string, string, *jwtError) {
	// read cookies
	if a.options.BearerTokens {
		// tokens are not in cookies
		// Note: we don't check for errors here, because we will check if the token is valid, later
		return r.Header.Get(a.options.AuthTokenName), r.Header.Get(a.options.RefreshTokenName), nil
	}

	var (
		authCookieValue    string
		refreshCookieValue string
	)

	AuthCookie, authErr := r.Cookie(a.options.AuthTokenName)
	if authErr == http.ErrNoCookie {
		a.myLog("Unauthorized attempt! No auth cookie")
		return "", "", newJwtError(errors.New("no auth cookie"), 401)
	} else if authErr != nil {
		// a.myLog(authErr)
		return "", "", newJwtError(errors.New("internal Server Error"), 500)
	}

	RefreshCookie, refreshErr := r.Cookie(a.options.RefreshTokenName)
	if refreshErr != nil && refreshErr != http.ErrNoCookie {
		a.myLog(refreshErr)
		return "", "", newJwtError(errors.New("internal Server Error"), 500)
	}

	if AuthCookie != nil {
		authCookieValue = AuthCookie.Value
	}
	if RefreshCookie != nil {
		refreshCookieValue = RefreshCookie.Value
	}

	return authCookieValue, refreshCookieValue, nil
}

func (a *Auth) extractCsrfStringFromReq(r *http.Request) (string, *jwtError) {
	// csrfCookie, _ := r.Cookie(a.options.CSRFTokenName)
	// if csrfCookie != nil {
	// 	return csrfCookie.Value, nil
	// }

	// csrfString := r.FormValue(a.options.CSRFTokenName)

	// if csrfString != "" {
	// 	return csrfString, nil
	// }

	csrfString := r.Header.Get(a.options.CSRFTokenName)
	if csrfString != "" {
		return csrfString, nil
	}

	auth := r.Header.Get("Authorization")
	csrfString = strings.Replace(auth, "Bearer", "", 1)
	csrfString = strings.Replace(csrfString, " ", "", -1)
	if csrfString == "" {
		return csrfString, newJwtError(errors.New("no CSRF string"), 401)
	}

	return csrfString, nil
}

func (a *Auth) setCredentialsOnResponseWriter(w http.ResponseWriter, c *credentials) *jwtError {
	var (
		refreshTokenString string
		refreshTokenClaims *ClaimsType
	)

	authTokenString, err := c.AuthToken.Token.SignedString(a.signKey)
	if err != nil {
		return newJwtError(err, 500)
	}
	if c.RefreshToken != nil && c.RefreshToken.Token != nil {
		a.myLog(c.RefreshToken)
		a.myLog(c.RefreshToken.Token)
		refreshTokenString, err = c.RefreshToken.Token.SignedString(a.signKey)
		if err != nil {
			return newJwtError(err, 500)
		}
	}

	if a.options.BearerTokens {
		// tokens are not in cookies
		setHeader(w, a.options.AuthTokenName, authTokenString)
		if refreshTokenString != "" {
			setHeader(w, a.options.RefreshTokenName, refreshTokenString)
		}
	} else {
		// tokens are in cookies
		// note: don't use an "Expires" in auth cookies bc browsers won't send expired cookies?
		authCookie := http.Cookie{
			Name:  a.options.AuthTokenName,
			Value: authTokenString,
			Path:  "/",
			// Expires:  time.Now().Add(a.options.AuthTokenValidTime),
			HttpOnly: true,
			Secure:   !a.options.IsDevEnv,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, &authCookie)

		if refreshTokenString != "" {
			refreshCookie := http.Cookie{
				Name:     a.options.RefreshTokenName,
				Value:    refreshTokenString,
				Expires:  time.Now().Add(a.options.RefreshTokenValidTime),
				Path:     "/",
				HttpOnly: true,
				Secure:   !a.options.IsDevEnv,
				SameSite: http.SameSiteStrictMode,
			}
			http.SetCookie(w, &refreshCookie)
		}

		// csrfCookie := http.Cookie{
		// 	Name:     a.options.CSRFTokenName,
		// 	Value:    c.CsrfString,
		// 	Expires:  time.Now().Add(a.options.RefreshTokenValidTime),
		// 	Path:     "/",
		// 	HttpOnly: true,
		// 	Secure:   true,
		// 	SameSite: http.SameSiteStrictMode,
		// }
		// http.SetCookie(w, &csrfCookie)

	}

	authTokenClaims, ok := c.AuthToken.Token.Claims.(*ClaimsType)
	if !ok {
		a.myLog("Cannot read auth token claims")
		return newJwtError(errors.New("cannot read token claims"), 500)
	}
	if c.RefreshToken != nil && c.RefreshToken.Token != nil {
		refreshTokenClaims, ok = c.RefreshToken.Token.Claims.(*ClaimsType)
		if !ok {
			a.myLog("Cannot read refresh token claims")
			return newJwtError(errors.New("cannot read token claims"), 500)
		}
	}

	w.Header().Set(a.options.CSRFTokenName, c.CsrfString)
	// note @adam-hanna: this may not be correct when using a sep auth server?
	//    							 bc it checks the request?
	w.Header().Set("Auth-Expiry", strconv.FormatInt(authTokenClaims.RegisteredClaims.ExpiresAt.Unix(), 10))
	if refreshTokenClaims != nil {
		w.Header().Set("Refresh-Expiry", strconv.FormatInt(refreshTokenClaims.RegisteredClaims.ExpiresAt.Unix(), 10))
	}

	return nil
}

func (a *Auth) buildCredentialsFromRequest(r *http.Request, c *credentials) *jwtError {
	authTokenString, refreshTokenString, err := a.extractTokenStringsFromReq(r)
	if err != nil {
		return newJwtError(err, 500)
	}

	csrfString, err := a.extractCsrfStringFromReq(r)
	if err != nil {
		return newJwtError(err, 500)
	}

	err = a.buildCredentialsFromStrings(csrfString, authTokenString, refreshTokenString, c)
	if err != nil {
		return newJwtError(err, 500)
	}

	return nil
}

func (a *Auth) myLog(stoofs interface{}) {
	if a.options.Debug {
		log.Println(stoofs)
	}
}

func setHeader(w http.ResponseWriter, header string, value string) {
	w.Header().Set(header, value)
}
