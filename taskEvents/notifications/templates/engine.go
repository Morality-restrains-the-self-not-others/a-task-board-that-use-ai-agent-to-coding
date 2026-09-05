package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

//go:embed *.txt *.html
var files embed.FS

var registeredTemplates = []string{
	"invitation",
	"password_reset",
	"activation",
	"verification_code",
	"welcome",
	"email_registration_invite",
}

type templatePair struct {
	text *texttemplate.Template
	html *template.Template
}

// Engine renders embedded notification templates.
type Engine struct {
	byName map[string]templatePair
}

func NewEngine() (*Engine, error) {
	byName := make(map[string]templatePair, len(registeredTemplates))
	for _, name := range registeredTemplates {
		txtRaw, err := files.ReadFile(name + ".txt")
		if err != nil {
			return nil, fmt.Errorf("read %s.txt: %w", name, err)
		}
		htmlRaw, err := files.ReadFile(name + ".html")
		if err != nil {
			return nil, fmt.Errorf("read %s.html: %w", name, err)
		}
		txtT, err := texttemplate.New(name + ".txt").Parse(string(txtRaw))
		if err != nil {
			return nil, fmt.Errorf("parse %s.txt: %w", name, err)
		}
		htmlT, err := template.New(name + ".html").Parse(string(htmlRaw))
		if err != nil {
			return nil, fmt.Errorf("parse %s.html: %w", name, err)
		}
		byName[name] = templatePair{text: txtT, html: htmlT}
	}
	return &Engine{byName: byName}, nil
}

// Render renders a named template with JSON-style snake_case context from EMAIL_SENT payloads.
func (e *Engine) Render(templateName string, context map[string]interface{}) (text string, html string, err error) {
	pair, ok := e.byName[templateName]
	if !ok {
		return "", "", fmt.Errorf("unknown template %q", templateName)
	}
	if context == nil {
		context = map[string]interface{}{}
	}
	data, err := mapContext(templateName, context)
	if err != nil {
		return "", "", err
	}
	var tb, hb bytes.Buffer
	if err = pair.text.Execute(&tb, data); err != nil {
		return "", "", fmt.Errorf("%s text: %w", templateName, err)
	}
	if err = pair.html.Execute(&hb, data); err != nil {
		return "", "", fmt.Errorf("%s html: %w", templateName, err)
	}
	return tb.String(), hb.String(), nil
}

func (e *Engine) RenderInvitation(data InvitationData) (text string, html string, err error) {
	return e.Render("invitation", map[string]interface{}{
		"company_name":     data.CompanyName,
		"invitation_url":   data.InvitationURL,
		"expiration_days":  data.ExpirationDays,
		"message":          data.Message,
		"unsubscribe_url":  data.UnsubscribeURL,
	})
}
