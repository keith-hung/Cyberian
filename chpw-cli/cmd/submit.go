package cmd

import (
	"github.com/keith-hung/chpw-cli/internal/types"
	"github.com/spf13/cobra"
)

var submitOtp string

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit the new password with the SMS OTP (run login first)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if submitOtp == "" {
			ExitError("--otp is required", 3)
		}
		if !gf.PassStdin || gf.Pass == "" {
			ExitError("new password required via --pass-stdin", 3)
		}
		c, err := newFlow()
		if err != nil {
			// newFlow also checks --pass-stdin/user/url; reuse its messages.
			ExitError(err.Error(), classifyError(err))
		}
		if err := c.Submit(gf.Pass, submitOtp); err != nil {
			ExitError(err.Error(), classifyError(err))
		}
		OutputJSON(types.SubmitOutput{Success: true, Message: "password changed"}, gf.Pretty)
		return nil
	},
}

func init() {
	submitCmd.Flags().StringVar(&submitOtp, "otp", "", "SMS one-time password (6 digits)")
	rootCmd.AddCommand(submitCmd)
}
