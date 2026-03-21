package model

import (
	"encoding/json"
	"errors"
)

// ChannelType identifies the delivery mechanism for an alert channel.
type ChannelType string

const (
	ChannelTypeWebhook ChannelType = "webhook"
	ChannelTypeEmail   ChannelType = "email"
	ChannelTypeSMS     ChannelType = "sms"
)

var validChannelTypes = map[ChannelType]struct{}{
	ChannelTypeWebhook: {},
	ChannelTypeEmail:   {},
	ChannelTypeSMS:     {},
}

// ChannelConfig is a user-configured alert delivery target.
type ChannelConfig struct {
	ID      string
	Type    ChannelType
	Config  json.RawMessage
	Enabled bool
}

// Validate returns an error if the ChannelConfig is not valid.
func (c ChannelConfig) Validate() error {
	if _, ok := validChannelTypes[c.Type]; !ok {
		return errors.New("channel type must be one of: webhook, email, sms")
	}
	return nil
}
