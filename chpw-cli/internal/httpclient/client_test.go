package httpclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
)

func TestGetFollowsRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		w.Write([]byte("landed"))
	}))
	defer srv.Close()

	c, err := New(srv.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get("start")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Body != "landed" {
		t.Errorf("body = %q, want %q", resp.Body, "landed")
	}
}

func TestSessionRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc", Path: "/"})
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, false)
	if _, err := c.Get(""); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "sess.json")
	if err := c.SaveSession(file, map[string]interface{}{"token": "T2"}); err != nil {
		t.Fatal(err)
	}

	c2, _ := New(srv.URL, false)
	info, err := c2.LoadSession(file)
	if err != nil {
		t.Fatal(err)
	}
	if info["token"] != "T2" {
		t.Errorf("token = %v, want T2", info["token"])
	}
	parsed, _ := url.Parse(srv.URL)
	found := false
	for _, ck := range c2.jar.Cookies(parsed) {
		if ck.Name == "sid" && ck.Value == "abc" {
			found = true
		}
	}
	if !found {
		t.Error("restored jar missing cookie sid=abc")
	}
}
