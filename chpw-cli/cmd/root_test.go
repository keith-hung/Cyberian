package cmd

import "testing"

func TestClassifyError(t *testing.T) {
	cases := map[string]int{
		"authentication failed: bad password": 2,
		"validation: 驗證碼錯誤":                   3,
		"POST : dial tcp: no such host":       4,
		"something unexpected":                1,
	}
	for msg, want := range cases {
		if got := classifyError(errString(msg)); got != want {
			t.Errorf("classifyError(%q) = %d, want %d", msg, got, want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
