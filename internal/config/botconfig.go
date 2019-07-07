package config

import "encoding/xml"

// IRCChan represents an IRC channel and an optional key
type IRCChan struct {
	Name string `xml:"name,attr"`
	Key  string `xml:"key,attr,omitempty"`
}

// NSAuth represents all the data required to login with NickServ
type NSAuth struct {
	// TODO: add require login bool
	Nick     string `xml:"nick,attr"`
	Password string `xml:"password,attr"`
	SASL     bool   `xml:"sasl,attr"`
}

// BotConfig represents the configuration options for a bot.Bot
type BotConfig struct {
	XMLName         xml.Name  `xml:"bot_config"`
	Nick            string    `xml:"nick,attr"`
	Ident           string    `xml:"ident,attr"`
	Gecos           string    `xml:"gecos,attr"`
	Host            string    `xml:"host,attr"`
	Port            string    `xml:"port,attr"`
	SSL             bool      `xml:"ssl,attr"`
	CommandPrefix   string    `xml:"command_prefix,attr"`
	AdminChan       IRCChan   `xml:"admin_chan"`
	ConnectCommands []string  `xml:"connect_commands>command,omitempty"`
	JoinChans       []IRCChan `xml:"autojoin_channels>channel,omitempty"`
	NSAuth          NSAuth    `xml:"auth>nickserv"`
}

// Permission represents an IRC mask that has additional access to the bot
type Permission struct {
	XMLName xml.Name `xml:"permission"`
	Mask    string   `xml:"mask,attr"`
}
