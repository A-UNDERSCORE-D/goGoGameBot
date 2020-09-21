// Package network implements a Transport that works over a socket.
package network

import (
	"encoding/xml"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/util"
)

// Config is a config for a networkTransport
type Config struct {
	util.BaseConfig
	Name       xml.Name `xml:"config"`
	Address    string   `xml:"address"`
	StartLocal bool     `xml:"start_local"`
	IsUnix     bool     `xml:"is_unix,attr"`
	TLS        bool     `xml:"tls,attr"`
}
