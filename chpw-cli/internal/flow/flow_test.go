package flow

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const loginHTML = `<form><input name="__RequestVerificationToken" value="TOK-LOGIN"/>
<input name="Username"/><input name="Password" type="password"/></form>`

const loginErrHTML = `<form><input name="__RequestVerificationToken" value="TOK-LOGIN"/>
<input name="Username"/><input name="Password" type="password"/>
<span class="field-validation-error" data-valmsg-for="ErrorMessage">帳號或密碼錯誤</span></form>`

const otpHTML = `<form><input name="__RequestVerificationToken" value="TOK-OTP"/>
<input name="NewPassword" type="password"/><input name="ConfirmPassword" type="password"/>
<input name="Otp" placeholder="請輸入簡訊驗證碼"/></form>`

const otpErrHTML = otpHTML + `<span class="field-validation-error" data-valmsg-for="Otp">驗證碼錯誤</span>`

const completeHTML = `<div>恭喜您！新密碼已修改完成！</div>`

// stubServer emulates the three portal endpoints.
func stubServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && (r.URL.Path == "/" || r.URL.Path == "/ChangePassword/Login"):
			http.SetCookie(w, &http.Cookie{Name: "__RequestVerificationToken", Value: "cookie", Path: "/"})
			w.Write([]byte(loginHTML))
		case r.Method == http.MethodPost && r.URL.Path == "/ChangePassword/Login":
			r.ParseForm()
			if r.PostForm.Get("Password") == "good" {
				w.Write([]byte(otpHTML))
			} else {
				w.Write([]byte(loginErrHTML))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/ChangePassword/Submit":
			r.ParseForm()
			if r.PostForm.Get("Otp") == "123456" && r.PostForm.Get("NewPassword") != "" {
				w.Write([]byte(completeHTML))
			} else {
				w.Write([]byte(otpErrHTML))
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func newClient(t *testing.T, url, sessFile string) *Client {
	c, err := New(Config{BaseURL: url, Username: "alice", SessionFile: sessFile})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoginSuccessPersistsSession(t *testing.T) {
	srv := stubServer(t)
	defer srv.Close()
	sess := filepath.Join(t.TempDir(), "s.json")

	res, err := newClient(t, srv.URL, sess).Login("good")
	if err != nil {
		t.Fatal(err)
	}
	if res.OtpTTL != 120 || res.SessionTTL != 180 {
		t.Errorf("ttl = %d/%d, want 120/180", res.OtpTTL, res.SessionTTL)
	}
	// Submit (fresh client) must load the persisted token + cookies and succeed.
	if err := newClient(t, srv.URL, sess).Submit("NewPass123!", "123456"); err != nil {
		t.Fatalf("submit after login: %v", err)
	}
}

func TestLoginBadPassword(t *testing.T) {
	srv := stubServer(t)
	defer srv.Close()
	_, err := newClient(t, srv.URL, filepath.Join(t.TempDir(), "s.json")).Login("wrong")
	if err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("err = %v, want authentication error", err)
	}
}

func TestSubmitWrongOtp(t *testing.T) {
	srv := stubServer(t)
	defer srv.Close()
	sess := filepath.Join(t.TempDir(), "s.json")
	if _, err := newClient(t, srv.URL, sess).Login("good"); err != nil {
		t.Fatal(err)
	}
	err := newClient(t, srv.URL, sess).Submit("NewPass123!", "000000")
	if err == nil || !strings.Contains(err.Error(), "validation") {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestSubmitWithoutLogin(t *testing.T) {
	srv := stubServer(t)
	defer srv.Close()
	err := newClient(t, srv.URL, filepath.Join(t.TempDir(), "missing.json")).Submit("NewPass123!", "123456")
	if err == nil {
		t.Fatal("expected error when no session exists")
	}
}
