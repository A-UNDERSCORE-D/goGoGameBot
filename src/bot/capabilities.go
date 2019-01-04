package bot

import (
    "fmt"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/util"
    "strings"
)

type CapabilityManager struct {
    capsToRequest []string
    capsAvailable []string
    capsAccepted  []string
    bot           *Bot
}

func (c *CapabilityManager) requestCap(requestedCap string) {
    c.capsToRequest = append(c.capsToRequest, requestedCap)
}

func (c *CapabilityManager) hasCap(requestedCap string) bool {
    for _, v := range c.capsAccepted {
        capName := v
        if strings.Contains(capName, "=") {
            capName = strings.Split(capName, "=")[0]
        }

        if requestedCap == v {
            return true
        }
    }
    return false
}

func (c *CapabilityManager) NegotiateCaps() {

    if len(c.capsToRequest) < 1 {
        return
    }

    lineChan, done := c.bot.GetRawChan("CAP")
    if err := c.bot.WriteLine(util.MakeSimpleIRCLine("CAP", "LS", "302")); err != nil {
        c.bot.Error(fmt.Errorf("could not negotiate capabilities: %s", err))
        return
    }
    defer func() { done <- true }()
    defer func() { c.bot.WriteLine(util.MakeSimpleIRCLine("CAP", "END")) }()



    loop:
    for line := range lineChan {
        switch line.Params[1] {
        case "LS":
            if line.Params[2] == "*" {
                // This is multiline, lets just append for now
                c.capsAvailable = append(c.capsAvailable, strings.Split(line.Params[3], " ")...)
                continue
            } else {
                c.capsAvailable = append(c.capsAvailable, strings.Split(line.Params[2], " ")...)
            }
            c.bot.Log.Printf("Server offered caps: %s", strings.Join(c.capsAvailable, " ,"))

            if err := c.requestCaps(); err != nil {
                c.bot.Error(fmt.Errorf("could not request caps: %s", err))
                return
            }

        case "ACK":
            c.capsAccepted = append(c.capsAccepted, strings.Split(line.Params[1], " ")...)
            break loop

        case "NAK":
            // To be implemented >.>
            panic("Not Implemented")
        }
    }

}

func (c *CapabilityManager) requestCaps() error {
    capsToReq := capIntersect(c.capsAvailable, c.capsToRequest)
    err := c.bot.WriteLine(
        util.MakeSimpleIRCLine("CAP", "REQ", strings.Join(capsToReq, " ")),
    )
    if err != nil {
        return err
    }
    return nil
}

func capIntersect(s1, s2 []string) []string {
    var out []string
    for _, v1 := range s1 {
        for _, v2 := range s2 {
            if v1 == v2 {
                out = append(out, v1)
            }
        }
    }
    return out
}
