package main

import (
	"errors"
	"net/rpc"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/AlertFlow/runner/pkg/executions"
	"github.com/AlertFlow/runner/pkg/plugins"

	"github.com/v1Flows/alertFlow/services/backend/pkg/models"

	"github.com/hashicorp/go-plugin"
)

// Plugin is an implementation of the Plugin interface
type Plugin struct{}

func (p *Plugin) ExecuteTask(request plugins.ExecuteTaskRequest) (plugins.Response, error) {
	from := ""
	password := ""
	to := []string{}
	smtpHost := ""
	smtpPort := 0
	message := ""

	for _, param := range request.Step.Action.Params {
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

	err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:        request.Step.ID,
		Messages:  []string{`Authenticate on SMTP Server: ` + smtpHost + `:` + strconv.Itoa(smtpPort)},
		StartedAt: time.Now(),
		Status:    "running",
	})
	if err != nil {
		return plugins.Response{
			Success: false,
		}, err
	}

	// Create authentication
	auth := smtp.PlainAuth("", from, password, smtpHost+":"+strconv.Itoa(smtpPort))

	// Send actual message
	err = smtp.SendMail(smtpHost+":"+strconv.Itoa(smtpPort), auth, from, to, []byte(message))
	if err != nil {
		err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID:         request.Step.ID,
			Messages:   []string{`Failed to send email: ` + err.Error()},
			Status:     "error",
			FinishedAt: time.Now(),
		})
		if err != nil {
			return plugins.Response{
				Success: false,
			}, err
		}

		return plugins.Response{
			Success: false,
		}, nil
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:         request.Step.ID,
		Messages:   []string{`Email sent to ` + strings.Join(to, ", ")},
		Status:     "success",
		FinishedAt: time.Now(),
	})
	if err != nil {
		return plugins.Response{
			Success: false,
		}, err
	}

	return plugins.Response{
		Success: true,
	}, nil
}

func (p *Plugin) HandleAlert(request plugins.AlertHandlerRequest) (plugins.Response, error) {
	return plugins.Response{
		Success: false,
	}, errors.New("not implemented")
}

func (p *Plugin) Info() (models.Plugins, error) {
	var plugin = models.Plugins{
		Name:    "Mail",
		Type:    "action",
		Version: "1.1.1",
		Author:  "JustNZ",
		Actions: models.Actions{
			Name:        "Mail",
			Description: "Send an email",
			Plugin:      "mail",
			Icon:        "solar:mailbox-linear",
			Category:    "Utility",
			Params: []models.Params{
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
					Default:     "587",
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
			},
		},
		Endpoints: models.AlertEndpoints{},
	}

	return plugin, nil
}

// PluginRPCServer is the RPC server for Plugin
type PluginRPCServer struct {
	Impl plugins.Plugin
}

func (s *PluginRPCServer) ExecuteTask(request plugins.ExecuteTaskRequest, resp *plugins.Response) error {
	result, err := s.Impl.ExecuteTask(request)
	*resp = result
	return err
}

func (s *PluginRPCServer) HandleAlert(request plugins.AlertHandlerRequest, resp *plugins.Response) error {
	result, err := s.Impl.HandleAlert(request)
	*resp = result
	return err
}

func (s *PluginRPCServer) Info(args interface{}, resp *models.Plugins) error {
	result, err := s.Impl.Info()
	*resp = result
	return err
}

// PluginServer is the implementation of plugin.Plugin interface
type PluginServer struct {
	Impl plugins.Plugin
}

func (p *PluginServer) Server(*plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

func (p *PluginServer) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &plugins.PluginRPC{Client: c}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "hello",
		},
		Plugins: map[string]plugin.Plugin{
			"plugin": &PluginServer{Impl: &Plugin{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
