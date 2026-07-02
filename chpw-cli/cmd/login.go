package cmd

import (
	"fmt"
	"strings"

	"github.com/keith-hung/chpw-cli/internal/flow"
	"github.com/keith-hung/chpw-cli/internal/types"
	"github.com/spf13/cobra"
)

var loginMethod string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Verify current credentials; the server sends an SMS OTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newFlow()
		if err != nil {
			ExitError(err.Error(), classifyError(err))
		}
		method := strings.ToUpper(strings.TrimSpace(loginMethod))
		if method != "APP" && method != "SMS" {
			ExitError("--method must be APP or SMS", 3)
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
				Command: "chpw submit --pass-stdin --otp <OTP>",
				Hint:    fmt.Sprintf("Pipe the NEW password to stdin; same directory, within %ds.", res.OtpTTL),
			},
		}, gf.Pretty)
		return nil
	},
}

// newFlow validates required inputs and builds a flow client.
func newFlow() (*flow.Client, error) {
	if gf.URL == "" {
		return nil, fmt.Errorf("config: --url or CHPW_BASE_URL is required")
	}
	if gf.User == "" {
		return nil, fmt.Errorf("config: --user or CHPW_USERNAME is required")
	}
	if !gf.PassStdin || gf.Pass == "" {
		return nil, fmt.Errorf("password required via --pass-stdin")
	}
	return flow.New(flow.Config{
		BaseURL:     gf.URL,
		Username:    gf.User,
		SessionFile: gf.SessionFile,
		Insecure:    gf.Insecure,
	})
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&loginMethod, "method", "APP", "OTP delivery method: APP (i-daka/Email) or SMS")
}
