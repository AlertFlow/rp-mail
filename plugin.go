package main

import (
	"encoding/json"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/AlertFlow/runner/pkg/executions"
	"github.com/AlertFlow/runner/pkg/models"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type EmailPlugin struct{}

func (p *EmailPlugin) Init() models.Plugin {
	return models.Plugin{
		Name:    "Email",
		Type:    "action",
		Version: "1.0.1",
		Creator: "JustNZ",
	}
}

func (p *EmailPlugin) Details() models.PluginDetails {
	params := []models.Param{
		{
			Key:         "From",
			Type:        "text",
			Default:     "from@mail.com",
			Required:    true,
			Description: "Sender email address",
		},
		{
			Key:         "Password",
			Type:        "password",
			Default:     "***",
			Required:    false,
			Description: "Sender email password",
		},
		{
			Key:         "To",
			Type:        "text",
			Default:     "to@mail.com",
			Required:    false,
			Description: "Recipient email address. Multiple emails can be separated by comma",
		},
		{
			Key:         "SmtpHost",
			Type:        "text",
			Default:     "smtp.mail.com",
			Required:    true,
			Description: "SMTP server host",
		},
		{
			Key:         "SmtpPort",
			Type:        "number",
			Default:     587,
			Required:    true,
			Description: "SMTP server port",
		},
		{
			Key:         "Message",
			Type:        "textarea",
			Default:     "Email message",
			Required:    true,
			Description: "Email message",
		},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Error(err)
	}

	return models.PluginDetails{
		Action: models.ActionDetails{
			ID:          "mail",
			Name:        "Mail",
			Description: "Sends an email",
			Icon:        "solar:mailbox-linear",
			Type:        "mail",
			Category:    "Utility",
			Function:    p.Execute,
			Params:      json.RawMessage(paramsJSON),
		},
	}
}

func (p *EmailPlugin) Execute(execution models.Execution, flow models.Flows, payload models.Payload, steps []models.ExecutionSteps, step models.ExecutionSteps, action models.Actions) (data map[string]interface{}, finished bool, canceled bool, no_pattern_match bool, failed bool) {
	from := ""
	password := ""
	to := []string{}
	smtpHost := ""
	smtpPort := 0
	message := ""

	for _, param := range action.Params {
		if param.Key == "From" {
			from = param.Value
		}
		if param.Key == "Password" {
			password = param.Value
		}
		if param.Key == "To" {
			to = strings.Split(param.Value, ",")
		}
		if param.Key == "SmtpHost" {
			smtpHost = param.Value
		}
		if param.Key == "SmtpPort" {
			smtpPort, _ = strconv.Atoi(param.Value)
		}
		if param.Key == "Message" {
			message = param.Value
		}
	}

	err := executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionID:       action.ID.String(),
		ActionMessages: []string{`Authenticate on SMTP Server: ` + smtpHost + `:` + strconv.Itoa(smtpPort)},
		Pending:        false,
		StartedAt:      time.Now(),
		Running:        true,
	})
	if err != nil {
		return nil, false, false, false, true
	}

	// Create authentication
	auth := smtp.PlainAuth("", from, password, smtpHost+":"+strconv.Itoa(smtpPort))

	// Send actual message
	err = smtp.SendMail(smtpHost+":"+strconv.Itoa(smtpPort), auth, from, to, []byte(message))
	if err != nil {
		err := executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
			ID:             step.ID,
			ActionID:       action.ID.String(),
			ActionMessages: []string{`Failed to send email: ` + err.Error()},
			Pending:        false,
			Finished:       true,
			FinishedAt:     time.Now(),
			Running:        false,
			Error:          true,
		})
		log.Error(err.Error())
		return nil, false, false, false, true
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{`Email sent to ` + strings.Join(to, ", ")},
		Running:        false,
		Finished:       true,
		FinishedAt:     time.Now(),
	})
	if err != nil {
		log.Error(err.Error())
		return nil, false, false, false, true
	}

	return nil, true, false, false, false
}

func (p *EmailPlugin) Handle(context *gin.Context) {}

var Plugin EmailPlugin
