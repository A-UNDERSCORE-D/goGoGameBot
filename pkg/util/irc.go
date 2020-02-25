package util

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"
)

// IRC SASL numerics
// noinspection ALL
const (
	//revive:disable:var-naming
	RPL_LOGGEDIN    = "900"
	RPL_LOGGEDOUT   = "901"
	RPL_NICKLOCKED  = "902"
	RPL_SASLSUCCESS = "903"
	RPL_SASLFAIL    = "904"
	RPL_SASLTOOLONG = "905"
	RPL_SASLABORTED = "906"
	RPL_SASLALREADY = "907"
	RPL_SASLMECHS   = "908"
	//revive:enable:var-naming
)

// MakeSimpleIRCLine is a helper function that creates an ircmsg.IrcMessage with no tags and no prefix.
func MakeSimpleIRCLine(command string, args ...string) ircmsg.IrcMessage {
	return ircmsg.MakeMessage(nil, "", command, args...)
}

// GenerateSASLString generates a base64 encoded string from the given parameters that can be used for
// SASL PLAIN authentication with an IRC server
func GenerateSASLString(nick, saslUsername, saslPasswd string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s\x00%s\x00%s", nick, saslUsername, saslPasswd)),
	)
}

var charMap = map[rune]string{'?': ".", '*': ".*"}
var cacheMutex sync.Mutex
var regexpCache = make(map[string]*regexp.Regexp)

// GlobToRegexp converts a mask glob string to a regexp that will only allow the wildcards * and ? to have any special
// meaning.
func GlobToRegexp(mask string) *regexp.Regexp {
	cacheMutex.Lock()
	re, ok := regexpCache[mask]
	cacheMutex.Unlock()
	if ok {
		return re
	}

	out := strings.Builder{}

	for _, c := range mask {
		toUse, ok := charMap[c]
		if ok {
			out.WriteString(toUse)
		} else {
			out.WriteString(regexp.QuoteMeta(string(c)))
		}
	}

	re = regexp.MustCompile(out.String())
	cacheMutex.Lock()
	regexpCache[mask] = re
	cacheMutex.Unlock()
	return re
}

// AnyMaskMatch returns whether or not the given match
func AnyMaskMatch(toCheck string, masks []string) bool {
	for _, mask := range masks {
		if GlobToRegexp(mask).MatchString(toCheck) {
			return true
		}
	}
	return false
}

// UserHost2Canonical returns the nickname!username@host representation of the given ircutils.UserHost
func UserHost2Canonical(uh ircutils.UserHost) string {
	out := strings.Builder{}
	out.WriteString(uh.Nick)
	if uh.User != "" {
		out.WriteRune('!')
		out.WriteString(uh.Host)
	}
	if uh.Host != "" {
		out.WriteRune('@')
		out.WriteString(uh.Host)
	}
	return out.String()
}
