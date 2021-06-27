package systemd

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed systemd.service
var systemdTemplateText string

//go:embed launchd.plist
var launchdTemplateText string

type EnvVar struct {
	Name  string
	Value string
}

type Unit struct {
	WorkingDir  string
	Description string
	CommandLine string
	Command     []string
	Home        string
	Env         []EnvVar
}

func (c *Unit) Write(w io.Writer, initType string) error {
	tpl, err := c.getTemplate(initType)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	if err := tpl.Execute(buf, c); err != nil {
		return err
	}

	if _, err := io.Copy(w, buf); err != nil {
		return err
	}

	return nil
}

func (c *Unit) getTemplate(initType string) (*template.Template, error) {
	var templateText string
	switch initType {
	case "systemd":
		templateText = systemdTemplateText
	case "launchd":
		templateText = launchdTemplateText
	default:
		return nil, fmt.Errorf("not supported: %q", initType)
	}

	tpl := template.New("init-template")
	tpl.Option("missingkey=error")

	t, err := tpl.Parse(templateText)
	if err != nil {
		return nil, err
	}

	return t, nil
}

