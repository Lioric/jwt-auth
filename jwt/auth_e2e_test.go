package jwt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func recoverHandler(next http.Handler) http.Handler {
	// this catches any errors and returns an internal server error to the client
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Recovered! Panic: %+v", err)
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func TestNoTokens(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    1 * time.Second,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if res.StatusCode/100 != 4 {
		t.Errorf("Expected unathorized (4xx), received: %d", res.StatusCode)
	}
}

func TestOptionsMethod(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    1 * time.Second,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	req, err := http.NewRequest("OPTIONS", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d", resp.StatusCode)
	}
}

func TestWithValidAuthToken(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    15 * time.Minute,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d", resp.StatusCode)
	}
}

type FakeResponse struct {
	t       *testing.T
	headers http.Header
	body    []byte
	status  int
}

func (r *FakeResponse) Header() http.Header {
	return r.headers
}

func (r *FakeResponse) Write(body []byte) (int, error) {
	r.body = body
	return len(body), nil
}

func (r *FakeResponse) WriteHeader(status int) {
	r.status = status
}

func (r *FakeResponse) Assert(status int, body string) {
	if r.status != status {
		r.t.Errorf("expected status %+v to equal %+v", r.status, status)
	}
	if string(r.body) != body {
		r.t.Errorf("expected body %+v to equal %+v", string(r.body), body)
	}
}

func TestProcessWithGrabTokenClaims(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    15 * time.Minute,
		CSRFTokenName:         "mycsrftoken",
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"
		claims.CustomClaims["Extra"] = "data"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	w := &FakeResponse{
		t:       t,
		headers: make(http.Header),
	}
	c, _err := a.Process(w, req)
	if _err != nil {
		t.Errorf("Couldn't getcredentials from request; Err: %v", _err)
	}

	role := c.CustomClaims["Role"]
	extra := c.CustomClaims["Extra"]
	if role != "user" && extra != "data" {
		t.Errorf("Expected 'user', received: %v, 'data', received: %v", role, extra)
	}
}

func TestWithExpiredAuthToken(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    1 * time.Second,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	// send the request
	// need to sleep to check expiry time differences
	duration := time.Duration(1100) * time.Millisecond // Pause
	time.Sleep(duration)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d", resp.StatusCode)
	}
}

func TestWithExpiredAuthTokenAndCustomName(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    1 * time.Second,
		AuthTokenName:         "myauthtoken",
		RefreshTokenName:      "myrefreshtoken",
		CSRFTokenName:         "mycsrftoken",
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	// send the request
	// need to sleep to check expiry time differences
	duration := time.Duration(1100) * time.Millisecond // Pause
	time.Sleep(duration)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, received: %d", resp.StatusCode)
	}
}

func TestWithExpiredTokens(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 10 * time.Millisecond,
		AuthTokenValidTime:    10 * time.Millisecond,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	// send the request
	// need to sleep to check expiry time differences
	duration := time.Duration(1100) * time.Millisecond // Pause
	time.Sleep(duration)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode/100 != 4 {
		t.Errorf("Expected status code 4xx, received: %d", resp.StatusCode)
	}
}

func TestWithRevokedRefreshToken(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    10 * time.Millisecond,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}
	revokedTokens = make(map[string]string)      // defined in credentials_unit_test.go
	a.SetRevokeTokenFunction(RevokeRefreshToken) // defined in credentials_unit_test.go
	a.SetCheckTokenIdFunction(CheckRefreshToken) // defined in credentials_unit_test.go

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))
	w := httptest.NewRecorder()
	a.NullifyTokens(w, req) // req has the cookies

	// send the request
	// need to sleep to check expiry time differences
	duration := time.Duration(1100) * time.Millisecond // Pause
	time.Sleep(duration)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode/100 != 4 {
		t.Errorf("Expected status code 4xx, received: %d", resp.StatusCode)
	}
}

func TestWithInvalidCSRFString(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		Debug:    false,
		IsDevEnv: true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		a.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, "wrongString")

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode/100 != 4 {
		t.Errorf("Expected status code 4xx, received: %d", resp.StatusCode)
	}
}

func TestWithInvalidSigningMethod(t *testing.T) {
	var a Auth
	authErr := New(&a, Options{
		SigningMethodString: "HS256",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    15 * time.Minute,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}
	var b Auth
	authErr = New(&b, Options{
		SigningMethodString: "HS384",
		HMACKey: []byte(`#5K+¥¼ƒ~ew{¦Z³(æðTÉ(©„²ÒP.¿ÓûZ’ÒGï–Š´Ãwb="=.!r.OÀÍšõgÐ€£`),
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    15 * time.Minute,
		Debug:                 false,
		IsDevEnv:              true,
	})
	if authErr != nil {
		t.Errorf("Failed to build jwt server; Err: %v", authErr)
	}

	ts := httptest.NewServer(recoverHandler(a.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))))
	defer ts.Close()

	as := httptest.NewServer(recoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsType{}
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["Role"] = "user"

		b.IssueNewTokens(w, &claims)
		fmt.Fprintln(w, "Hello, client")
	})))
	defer as.Close()

	res, err := http.Get(as.URL)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}
	rc := res.Cookies()
	var authCookieIndex int
	var refreshCookieIndex int

	for i, cookie := range rc {
		if cookie.Name == a.options.AuthTokenName {
			authCookieIndex = i
		}
		if cookie.Name == a.options.RefreshTokenName {
			refreshCookieIndex = i
		}
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Errorf("Couldn't build request; Err: %v", err)
	}

	req.AddCookie(rc[authCookieIndex])
	req.AddCookie(rc[refreshCookieIndex])
	req.Header.Add(a.options.CSRFTokenName, res.Header.Get(a.options.CSRFTokenName))

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Couldn't send request to test server; Err: %v", err)
	}

	if resp.StatusCode/100 != 4 {
		t.Errorf("Expected status code 4xx, received: %d", resp.StatusCode)
	}
}

// test bearer tokens?
