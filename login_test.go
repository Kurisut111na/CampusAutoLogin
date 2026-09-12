package main

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

// TestSanitizeURLForLog 确保凭据参数不落日志
func TestSanitizeURLForLog(t *testing.T) {
	cases := []struct {
		name, in, wantContain, wantAbsent string
	}{
		{"portal密码", "http://10.0.1.5:801/eportal/portal/login?user_account=a&user_password=c2VjcmV0MDA=&x=1", "user_password=***", "c2VjcmV0MDA="},
		{"oldAPI明文密码", "http://10.0.1.5:80/drcom/login?DDDDD=2025xxx&upass=mypass123&0MKKey=123456", "upass=***", "mypass123"},
		{"无凭据URL原样", "http://1.2.3.4/a79.htm?wlanuserip=10.0.0.1", "wlanuserip=10.0.0.1", "***"},
	}
	for _, c := range cases {
		got := sanitizeURLForLog(c.in)
		if !strings.Contains(got, c.wantContain) {
			t.Errorf("%s: 输出应包含 %q, 实际: %s", c.name, c.wantContain, got)
		}
		if c.wantAbsent != "" && strings.Contains(got, c.wantAbsent) {
			t.Errorf("%s: 输出不应泄漏 %q, 实际: %s", c.name, c.wantAbsent, got)
		}
	}
}

// TestQueryEscapeBase64Password 含 + / = 的 base64 密码必须被转义，
// 否则服务端按标准 query 解码时 + 会变成空格，登录静默失败。
func TestQueryEscapeBase64Password(t *testing.T) {
	for _, pw := range []string{"a+b/c=d", "p+w", "纯中文密码", "pass&word=1"} {
		enc := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(pw)))
		decoded, err := url.QueryUnescape(enc)
		if err != nil || decoded != base64.StdEncoding.EncodeToString([]byte(pw)) {
			t.Errorf("密码 %q 转义往返失败: enc=%q decoded=%q err=%v", pw, enc, decoded, err)
		}
		if strings.Contains(enc, "+") && !strings.Contains(enc, "%2B") {
			t.Errorf("密码 %q 的转义结果含裸 +, 服务端会解成空格: %q", pw, enc)
		}
	}
}

// TestParseLoginResponse 覆盖 JSONP 结果解析的主要分支
func TestParseLoginResponse(t *testing.T) {
	cases := []struct {
		name string
		body string
		want LoginStatus
	}{
		{"成功", `dr1004({"result":1,"msg":""})`, LoginSuccess},
		{"成功字符串结果", `dr1004({"result":"1","msg":""})`, LoginSuccess},
		{"拒绝", `dr1004({"result":0,"msg":"password error"})`, LoginFailed},
		{"已在线msga", `dr1004({"result":0,"msga":"already online","msg":""})`, LoginAlreadyOnline},
		{"已在线msg=1", `dr1004({"result":0,"msg":1})`, LoginAlreadyOnline},
		{"captive portal拦截", "<!DOCTYPE html><html>login page</html>", LoginFailed},
		{"非JSON垃圾", `garbage without braces`, LoginFailed},
	}
	for _, c := range cases {
		got, _ := parseLoginResponse(200, []byte(c.body))
		if got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// 非 200 一律失败
	if got, _ := parseLoginResponse(302, []byte(`{"result":1}`)); got != LoginFailed {
		t.Errorf("非200应返回 LoginFailed, got %v", got)
	}
}
