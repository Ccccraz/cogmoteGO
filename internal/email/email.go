package email

import (
	"net/http"
	"strings"

	"github.com/Ccccraz/cogmoteGO/internal/keyring"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wneessen/go-mail"
)

type emailPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type emailConfig struct {
	From       string
	Password   string
	Host       string
	Port       int
	Recipients []string
}

type handlerError struct {
	status      int
	userMessage string
	logMessage  string
	err         error
	fields      []any
}

func (e *handlerError) logFields() []any {
	if e == nil {
		return nil
	}

	fields := make([]any, 0, len(e.fields)+2)
	fields = append(fields, e.fields...)
	if e.err != nil {
		fields = append(fields, "err", e.err)
	}

	return fields
}

func handleError(c *gin.Context, err *handlerError) bool {
	if err == nil {
		return false
	}

	fields := err.logFields()
	if len(fields) > 0 {
		logger.Logger.Error(err.logMessage, fields...)
	} else {
		logger.Logger.Error(err.logMessage)
	}

	c.JSON(err.status, gin.H{"error": err.userMessage})
	return true
}

func PostEmailHandler(c *gin.Context) {
	payload, err := parseEmailPayload(c)
	if handleError(c, err) {
		return
	}

	cfg, err := loadEmailConfig()
	if handleError(c, err) {
		return
	}

	message, err := buildEmailMessage(cfg, payload)
	if handleError(c, err) {
		return
	}

	if err := deliverEmail(cfg, message); handleError(c, err) {
		return
	}

	c.Status(http.StatusCreated)
}

func parseEmailPayload(c *gin.Context) (emailPayload, *handlerError) {
	var payload emailPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		return emailPayload{}, &handlerError{
			status:      http.StatusBadRequest,
			userMessage: "invalid email payload",
			logMessage:  "invalid email request payload",
			err:         err,
		}
	}

	payload.Subject = strings.TrimSpace(payload.Subject)
	payload.Body = strings.TrimSpace(payload.Body)

	if payload.Subject == "" {
		return emailPayload{}, &handlerError{
			status:      http.StatusBadRequest,
			userMessage: "email subject cannot be empty",
			logMessage:  "email subject is empty",
		}
	}

	if payload.Body == "" {
		return emailPayload{}, &handlerError{
			status:      http.StatusBadRequest,
			userMessage: "email body cannot be empty",
			logMessage:  "email body is empty",
		}
	}

	return payload, nil
}

func loadEmailConfig() (emailConfig, *handlerError) {
	emailSection := viper.Sub("email")
	if emailSection == nil {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "email configuration not found",
			logMessage:  "email configuration section missing",
		}
	}

	sendEmail := strings.TrimSpace(emailSection.GetString("send_email"))
	if sendEmail == "" {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "send_email not configured",
			logMessage:  "send_email is not configured",
		}
	}

	smtpHost := strings.TrimSpace(emailSection.GetString("smtp_host"))
	if smtpHost == "" {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "smtp_host not configured",
			logMessage:  "smtp_host is not configured",
		}
	}

	smtpPort := emailSection.GetInt("smtp_port")
	if smtpPort <= 0 {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "smtp_port not configured",
			logMessage:  "smtp_port is not configured or invalid",
			fields:      []any{"value", smtpPort},
		}
	}

	rawRecipients := emailSection.GetStringSlice("send_email_to")
	recipients := make([]string, 0, len(rawRecipients))
	for _, recipient := range rawRecipients {
		recipient = strings.TrimSpace(recipient)
		if recipient != "" {
			recipients = append(recipients, recipient)
		}
	}
	if len(recipients) == 0 {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "send_email_to not configured",
			logMessage:  "send_email_to is not configured",
		}
	}

	password, err := keyring.GetPassword(sendEmail)
	if err != nil {
		return emailConfig{}, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "email password not found",
			logMessage:  "failed to retrieve email password",
			err:         err,
		}
	}

	return emailConfig{
		From:       sendEmail,
		Password:   password,
		Host:       smtpHost,
		Port:       smtpPort,
		Recipients: recipients,
	}, nil
}

func buildEmailMessage(cfg emailConfig, payload emailPayload) (*mail.Msg, *handlerError) {
	message := mail.NewMsg()
	if err := message.From(cfg.From); err != nil {
		return nil, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "failed to prepare email",
			logMessage:  "failed to set email sender",
			err:         err,
		}
	}

	if err := message.To(cfg.Recipients...); err != nil {
		return nil, &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "failed to prepare email",
			logMessage:  "failed to set email recipient",
			err:         err,
		}
	}

	message.Subject(payload.Subject)
	message.SetBodyString(mail.TypeTextPlain, payload.Body)
	return message, nil
}

func deliverEmail(cfg emailConfig, message *mail.Msg) *handlerError {
	client, err := mail.NewClient(
		cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(cfg.From),
		mail.WithPassword(cfg.Password),
	)
	if err != nil {
		return &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "failed to send email",
			logMessage:  "failed to create smtp client",
			err:         err,
		}
	}

	if err := client.DialAndSend(message); err != nil {
		return &handlerError{
			status:      http.StatusInternalServerError,
			userMessage: "failed to send email",
			logMessage:  "failed to send email",
			err:         err,
		}
	}

	return nil
}

func RegisterRoutes(r gin.IRouter) {
	r.POST("/email", PostEmailHandler)
}
