package parser

import "testing"

const loginPage = `<form class="adweb-form">
<input name="__RequestVerificationToken" type="hidden" value="TOK-LOGIN" />
<input class="input100" name="Username" placeholder="請輸入帳號" type="text" value="" />
<input class="input100" name="Password" placeholder="請輸入密碼" type="password" />
<span class="adweb-error-text field-validation-valid" data-valmsg-for="ErrorMessage"></span>
</form>`

const loginErrorPage = `<form class="adweb-form">
<input name="__RequestVerificationToken" type="hidden" value="TOK-LOGIN" />
<input name="Username" value="" />
<input name="Password" type="password" />
<span class="field-validation-error adweb-error-text" data-valmsg-for="ErrorMessage">帳號或密碼錯誤</span>
</form>`

const otpPage = `<form class="adweb-form">
<input name="__RequestVerificationToken" type="hidden" value="TOK-OTP" />
<input name="NewPassword" type="password" />
<input name="ConfirmPassword" type="password" />
<input class="input100" name="Otp" placeholder="請輸入簡訊驗證碼" type="text" value="" />
</form>`

const completePage = `<div class="adweb-form"><div class="congratulation-text">恭喜您！<br/>新密碼已修改完成！</div></div>`

func TestParseAntiForgeryToken(t *testing.T) {
	if got := ParseAntiForgeryToken(otpPage); got != "TOK-OTP" {
		t.Errorf("token = %q, want TOK-OTP", got)
	}
}

func TestPageDetection(t *testing.T) {
	if !IsLoginPage(loginPage) {
		t.Error("loginPage should be detected as login page")
	}
	if !IsOtpPage(otpPage) {
		t.Error("otpPage should be detected as OTP page")
	}
	if IsOtpPage(loginPage) {
		t.Error("loginPage must not be detected as OTP page")
	}
	if !IsCompletePage(completePage) {
		t.Error("completePage should be detected as complete")
	}
}

func TestParseErrorMessage(t *testing.T) {
	if got := ParseErrorMessage(loginErrorPage); got != "帳號或密碼錯誤" {
		t.Errorf("error = %q, want 帳號或密碼錯誤", got)
	}
	if got := ParseErrorMessage(loginPage); got != "" {
		t.Errorf("clean login page error = %q, want empty", got)
	}
}
