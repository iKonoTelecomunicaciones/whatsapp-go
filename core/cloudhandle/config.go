package cloudhandle

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	up "go.mau.fi/util/configupgrade"
	"gopkg.in/yaml.v3"
)

//go:embed example-config.yaml
var ExampleConfig string

type umConfig Config

type DefaultPowerLevels struct {
	EventsDefault *int `yaml:"events_default"`
	UsersDefault  *int `yaml:"users_default"`
	StateDefault  *int `yaml:"state_default"`
	Redact        *int `yaml:"redact"`
	Ban           *int `yaml:"ban"`
	Invite        *int `yaml:"invite"`
	Kick          *int `yaml:"kick"`
}

type DefaultEventsLevels struct {
	Reaction       *int `yaml:"reaction"`
	RoomName       *int `yaml:"room_name"`
	RoomAvatar     *int `yaml:"room_avatar"`
	RoomTopic      *int `yaml:"room_topic"`
	RoomEncryption *int `yaml:"room_encryption"`
	RoomTombstone  *int `yaml:"room_tombstone"`
}

type Config struct {
	OSName string `yaml:"os_name"`

	DisplaynameTemplate        string `yaml:"displayname_template"`
	DisableStatusBroadcastSend bool   `yaml:"disable_status_broadcast_send"`

	displaynameTemplate *template.Template `yaml:"-"`

	DefaultPowerLevels  *DefaultPowerLevels  `yaml:"default_power_levels"`
	DefaultEventsLevels *DefaultEventsLevels `yaml:"default_events_levels"`
	DefaultUserLevel    int                  `yaml:"default_user_level"`
}

// UnmarshalYAML customizes the YAML unmarshalling for Config.
// It decodes the YAML node into the Config struct and then runs post-processing logic.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	err := node.Decode((*umConfig)(c))
	if err != nil {
		return err
	}
	return c.PostProcess()
}

// PostProcess parses the display name template string and stores the compiled template.
// This allows the template to be used efficiently at runtime.
func (c *Config) PostProcess() error {
	var err error
	c.displaynameTemplate, err = template.New("displayname").Parse(c.DisplaynameTemplate)
	return err
}

// upgradeConfig helps migrate or upgrade configuration fields to new versions.
// It copies relevant fields from the old config to the new config during upgrades.
func upgradeConfig(helper up.Helper) {
	helper.Copy(up.Str, "os_name")

	helper.Copy(up.Str, "displayname_template")

	helper.Copy(up.Bool, "disable_status_broadcast_send")
}

type DisplaynameParams struct {
	types.ContactInfo
	Phone string
}

// FormatDisplayname generates a display name using the configured template.
// It fills the template with the provided JID, phone, and contact information.
func (c *Config) FormatDisplayname(jid string, phone string, contact types.ContactInfo) string {
	var nameBuf strings.Builder
	if phone == "" {
		phone = "+" + jid
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
