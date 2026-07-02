package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/chpw-cli/internal/flow"
	"github.com/keith-hung/chpw-cli/internal/types"
	"golang.org/x/term"
)

func validateMethod() string {
	m := strings.ToUpper(strings.TrimSpace(fMethod))
	if m != "APP" && m != "SMS" {
		ExitError("--method must be APP or SMS", 3)
	}
	return m
}

func requireURL() {
	if gf.URL == "" {
		ExitError("config: --url or CHPW_BASE_URL is required", 2)
	}
}

// runStart is step 1: verify current password, trigger the OTP, persist the
// session, and print the next-step command.
func runStart() {
	requireURL()
	if gf.User == "" {
		ExitError("config: --user or CHPW_USERNAME is required", 2)
	}
	if !gf.PassStdin || gf.Pass == "" {
		ExitError("current password required via --pass-stdin", 3)
	}
	method := validateMethod()
	c, err := flow.New(flow.Config{BaseURL: gf.URL, Username: gf.User, SessionFile: gf.SessionFile, Insecure: gf.Insecure})
	if err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	res, err := c.Login(gf.Pass, method)
	if err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	OutputJSON(types.LoginOutput{
		Success:    true,
		Message:    res.Message,
		OtpTTL:     res.OtpTTL,
		SessionTTL: res.SessionTTL,
		Next: types.NextStep{
			Command: "chpw --continue --pass-stdin --otp <OTP>",
			Hint:    fmt.Sprintf("Pipe the NEW password to stdin; same directory, within %ds.", res.OtpTTL),
		},
	}, gf.Pretty)
}

// runContinue is step 2: submit the new password + OTP using the persisted session.
func runContinue() {
	requireURL()
	if fOtp == "" {
		ExitError("--otp is required with --continue", 3)
	}
	if !gf.PassStdin || gf.Pass == "" {
		ExitError("new password required via --pass-stdin", 3)
	}
	c, err := flow.New(flow.Config{BaseURL: gf.URL, SessionFile: gf.SessionFile, Insecure: gf.Insecure})
	if err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	if err := c.Submit(gf.Pass, fOtp); err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	OutputJSON(types.SubmitOutput{Success: true, Message: "password changed"}, gf.Pretty)
}

// runInteractive is the -i one-shot: prompt for everything and change in one process.
func runInteractive() {
	if gf.PassStdin || fContinue || fOtp != "" {
		ExitError("--interactive cannot be combined with --pass-stdin, --continue, or --otp", 3)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		ExitError("--interactive requires a terminal; for non-interactive use omit -i (two-step: run once, then --continue)", 3)
	}
	requireURL()
	method := validateMethod()
	user := gf.User
	if user == "" {
		def := os.Getenv("USER")
		user = promptLine(fmt.Sprintf("Username [%s]: ", def))
		if user == "" {
			user = def
		}
		if user == "" {
			ExitError("config: username is required", 2)
		}
	}
	oldPass := promptPassword("Current password: ")
	if oldPass == "" {
		ExitError("current password is required", 3)
	}
	c, err := flow.New(flow.Config{BaseURL: gf.URL, Username: user, SessionFile: "", Insecure: gf.Insecure})
	if err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	res, err := c.Login(oldPass, method)
	if err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	fmt.Fprintln(os.Stderr, res.Message)
	otp := promptLine("Enter OTP: ")
	if otp == "" {
		ExitError("OTP is required", 3)
	}
	newPass := promptPassword("New password: ")
	confirm := promptPassword("Confirm new password: ")
	if newPass == "" {
		ExitError("new password is required", 3)
	}
	if newPass != confirm {
		ExitError("new passwords do not match", 3)
	}
	if err := c.Submit(newPass, otp); err != nil {
		ExitError(err.Error(), classifyError(err))
	}
	OutputJSON(types.SubmitOutput{Success: true, Message: "password changed"}, gf.Pretty)
}

// promptPassword reads a line with echo disabled (prompt to stderr).
func promptPassword(label string) string {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		ExitError(fmt.Sprintf("reading input: %v", err), 1)
	}
	return strings.TrimSpace(string(b))
}

// promptLine reads a plain line byte-by-byte (no buffering, so it is safe to
// interleave with term.ReadPassword on the same stdin fd).
func promptLine(label string) string {
	fmt.Fprint(os.Stderr, label)
	var sb strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			if buf[0] != '\r' {
				sb.WriteByte(buf[0])
			}
		}
		if err != nil {
			break
		}
	}
	return strings.TrimSpace(sb.String())
}
