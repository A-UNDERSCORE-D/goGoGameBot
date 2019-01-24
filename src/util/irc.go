package util

import (
    "encoding/base64"
    "fmt"
    "github.com/goshuirc/irc-go/ircmsg"
    "regexp"
    "strings"
)

//noinspection ALL
const (
    RPL_LOGGEDIN    = "900"
    RPL_LOGGEDOUT   = "901"
    RPL_NICKLOCKED  = "902"
    RPL_SASLSUCCESS = "903"
    RPL_SASLFAIL    = "904"
    RPL_SASLTOOLONG = "905"
    RPL_SASLABORTED = "906"
    RPL_SASLALREADY = "907"
    RPL_SASLMECHS   = "908"
)

func MakeSimpleIRCLine(command string, args ...string) ircmsg.IrcMessage {
    return ircmsg.MakeMessage(nil, "", command, args...)
}

func GenerateSASLString(nick, saslUsername, saslPasswd string) string {
    return base64.StdEncoding.EncodeToString(
        []byte(fmt.Sprintf("%s\x00%s\x00%s\x00", nick, saslUsername, saslPasswd)),
    )
}

var charMap = map[rune]string{'?': ".", '*': ".*"}
var regexpCache = make(map[string]*regexp.Regexp)

// GlobToRegexp converts a mask glob string to a regexp that will only allow the wildcards * and ? to have any special
// meaning.
func GlobToRegexp(mask string) *regexp.Regexp {
    re, ok := regexpCache[mask]
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
    regexpCache[mask] = re
    return re
}

func AnyMaskMatch(tocheck string, masks []string) bool {
    for _, mask := range masks {
        if GlobToRegexp(mask).MatchString(tocheck) {
            return true
        }
    }
    return false
}

func AddZwsp(s string) string {
    if len(s) < 2 {
        return s
    }

    return fmt.Sprintf("%c\u200b%s", s[0], s[1:])
}
