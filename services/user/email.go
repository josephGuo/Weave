package user

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"math/big"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
)

// EmailConfig 邮件服务器配置
type EmailConfig struct {
	SMTPServer string
	SMTPPort   int
	Username   string
	Password   string
	From       string
}

// emailer 邮件发送器（内部使用）
type emailer struct {
	config EmailConfig
}

// newEmailer 创建新的邮件发送器
func newEmailer(config EmailConfig) *emailer {
	return &emailer{config: config}
}

// isValidEmail 验证邮箱地址格式是否正确
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// generateVerificationCode 生成6位数字验证码
func (e *emailer) generateVerificationCode() (string, error) {
	result := make([]string, 6)
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = strconv.Itoa(int(num.Int64()))
	}
	return strings.Join(result, ""), nil
}

// sendVerificationCode 发送验证码到指定邮箱
func (e *emailer) sendVerificationCode(email, code string) error {
	if !isValidEmail(email) {
		return fmt.Errorf("invalid email address format")
	}

	subject := "Weave 验证码"

	// 对验证码进行HTML转义，防止XSS风险
	escapedCode := template.HTMLEscapeString(code)

	body := loadEmailTemplate(escapedCode)

	return e.sendEmail(email, subject, body)
}

// loadEmailTemplate 加载邮件模板（模板已内嵌到代码中）
func loadEmailTemplate(code string) string {
	return strings.Replace(emailTemplate, "{{.Code}}", code, -1)
}

// sendEmail 发送邮件
func (e *emailer) sendEmail(to, subject, body string) error {
	header := make(map[string]string)
	header["From"] = e.config.From
	header["To"] = to
	header["Subject"] = "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPServer)
	serverAddr := fmt.Sprintf("%s:%d", e.config.SMTPServer, e.config.SMTPPort)

	return smtp.SendMail(serverAddr, auth, e.config.From, []string{to}, []byte(message))
}

// emailTemplate 内嵌的邮件HTML模板
const emailTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Weave 登录验证码</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            padding: 30px;
        }
        h2 {
            color: #007bff;
            margin-top: 0;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
        }
        .code-box {
            font-size: 28px;
            font-weight: bold;
            color: #007bff;
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            padding: 15px;
            text-align: center;
            margin: 20px 0;
            letter-spacing: 3px;
        }
        p {
            margin: 10px 0;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 2px solid #f0f0f0;
            font-size: 14px;
            color: #6c757d;
        }
        .warning {
            color: #dc3545;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>您的验证码</h2>
        <p>尊敬的用户：</p>
        <p>您正在登录 Weave 系统，验证码为：</p>
        <div class="code-box">{{.Code}}</div>
        <p>该验证码有效期为5分钟，请尽快使用。</p>
        <p class="warning">请勿将验证码泄露给他人。</p>
        <p>如果您没有尝试登录，请忽略此邮件。</p>
        <div class="footer">
            <p>此致<br>Weave 团队</p>
        </div>
    </div>
</body>
</html>`
