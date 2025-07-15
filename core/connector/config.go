package connector

import (
	_ "embed"
	"strings"
	"text/template"

	up "go.mau.fi/util/configupgrade"
	"go.mau.fi/whatsmeow/types"
	"gopkg.in/yaml.v3"
)

type MediaRequestMethod string

const (
	MediaRequestMethodImmediate MediaRequestMethod = "immediate"
	MediaRequestMethodLocalTime MediaRequestMethod = "local_time"
)

//go:embed example-config.yaml
var ExampleConfig string

type Config struct {
	OSName string `yaml:"os_name"`

	DisplaynameTemplate        string `yaml:"displayname_template"`
	DisableStatusBroadcastSend bool   `yaml:"disable_status_broadcast_send"`

	displaynameTemplate *template.Template `yaml:"-"`
}

type umConfig Config

func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	err := node.Decode((*umConfig)(c))
	if err != nil {
		return err
	}
	return c.PostProcess()
}

func (c *Config) PostProcess() error {
	var err error
	c.displaynameTemplate, err = template.New("displayname").Parse(c.DisplaynameTemplate)
	return err
}

func upgradeConfig(helper up.Helper) {
	helper.Copy(up.Str, "os_name")

	helper.Copy(up.Str, "displayname_template")

	helper.Copy(up.Bool, "disable_status_broadcast_send")
}

type DisplaynameParams struct {
	types.ContactInfo
	Phone string
}

func (c *Config) FormatDisplayname(jid types.JID, phone string, contact types.ContactInfo) string {
	var nameBuf strings.Builder
	if phone == "" {
		phone = "+" + jid.User
		if jid.Server != types.DefaultUserServer {
			phone = jid.User
		}
	}
	err := c.displaynameTemplate.Execute(&nameBuf, &DisplaynameParams{
		ContactInfo: contact,
		Phone:       phone,
	})
	if err != nil {
		panic(err)
	}
	return nameBuf.String()
}

func (wa *WhatsappCloudConnector) GetConfig() (string, any, up.Upgrader) {
	return ExampleConfig, &wa.Config, &up.StructUpgrader{
		SimpleUpgrader: up.SimpleUpgrader(upgradeConfig),
		Blocks: [][]string{
			{"displayname_template"},
		},
		Base: ExampleConfig,
	}
}
