package bot

import (
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/util"
    "github.com/goshuirc/irc-go/ircmsg"
    "strings"
    "sync"
)

// Capability represents an IRCv3 capability
type Capability struct {
    Name      string // The name of the capability
    Params    string // The parameters passed back by the server after negotiation
    requested bool   // Whether or not the capability was requested by the bot
    available bool   // Whether or not the capability is available on the server
    enabled   bool   // whether or not the capability has been requested and acknowledged by the server

    Callback func(*Capability, ircmsg.IrcMessage, *sync.WaitGroup) // a callback to be called if the cap is accepted
}

func (c *Capability) String() string {
    if c.Params == "" {
        return c.Name
    }

    return fmt.Sprintf("%s: %s", c.Name, c.Params)
}

// CapabilityManager manages IRCv3 capabilities and can negotiate capabilities with an IRC server
type CapabilityManager struct {
    capabilities []*Capability
    bot          *Bot
}

// requestCap adds a Capability to the request list
func (cm *CapabilityManager) requestCap(capability *Capability) {
    capability.requested = true
    cm.capabilities = append(cm.capabilities, capability)
}

// getCapByName returns the Capability under the given name or a nil pointer if it does not exist
func (cm *CapabilityManager) getCapByName(name string) *Capability {
    for _, ca := range cm.capabilities {
        if ca.Name == name {
            return ca
        }
    }
    return nil
}

// hasCap returns whether or not the given cap name exists on the manager and if so, if it is marked as available
func (cm *CapabilityManager) hasCap(requestedCap string) bool {
    if c := cm.getCapByName(requestedCap); c != nil {
        return c.available
    }
    return false
}

// filterCaps returns all Capabilities that match the given predicate
func (cm *CapabilityManager) filterCaps(f func(*Capability) bool) []*Capability {
    var out []*Capability
    for _, c := range cm.capabilities {
        if f(c) {
            out = append(out, c)
        }
    }
    return out
}

// NegotiateCaps runs a IRCv3.2 cap negotiation sequence with the connected IRC server
func (cm *CapabilityManager) NegotiateCaps() {
    if len(cm.capabilities) < 1 {
        return
    }

    lineChan, done := cm.bot.GetRawChan("CAP")
    if err := cm.bot.WriteLine(util.MakeSimpleIRCLine("CAP", "LS", "302")); err != nil {
        cm.bot.Error(fmt.Errorf("could not negotiate capabilities: %s", err))
        return
    }
    wg := sync.WaitGroup{}

    for line := range lineChan {
        switch line.Params[1] {
        case "LS":
            cm.addAvailableCapability(line)
            wg.Add(1)

            if line.Params[2] == "*" {
                continue // This is multiline, lets just append for now
            }

            // we can setup cleanup things now. lets do that
            go func() {
                // Allow other things to run, then close up and send cap end
                wg.Wait()
                close(done)
                cm.bot.WriteLine(util.MakeSimpleIRCLine("CAP", "END"))
            }()

            cm.bot.Log.Infof("Server offered caps: %v",
                cm.filterCaps(func(capability *Capability) bool { return capability.available }),
            )

            if err := cm.requestCaps(); err != nil {
                cm.bot.Error(fmt.Errorf("could not request caps: %s", err))
                return
            }

        case "ACK":
            for _, v := range strings.Split(line.Params[2], " ") {
                name, params := parseCapStr(v)
                if c := cm.getCapByName(name); c != nil {
                    c.enabled = true
                    if c.Callback != nil {
                        wg.Add(1)
                        go c.Callback(c, line, &wg)
                    }
                } else {
                    cm.bot.Log.Warnf(
                        "Got an ACK for a cap we dont know about: %q with params %q", name, params,
                    )
                }
            }
            cm.bot.Log.Infof("Server ACK-ed caps: %v", cm.filterCaps(func(c *Capability) bool { return c.enabled }))
            wg.Done()

        case "NAK":
            // To be implemented >.>
            panic("Not Implemented")
        }
    }
}

// addAvailableCapability marks a capability as available if it exists on the cap list, otherwise it adds the cap and
// marks it as available but not requested
func (cm *CapabilityManager) addAvailableCapability(line ircmsg.IrcMessage) {
    for _, capStr := range strings.Split(line.Params[len(line.Params)-1], " ") {
        name, params := parseCapStr(capStr)
        if c := cm.getCapByName(name); c != nil {
            c.available = true
            c.Params = params
        } else {
            cm.capabilities = append(
                cm.capabilities,
                &Capability{Name: name, Params: params, requested: false, available: true},
            )
        }
    }
}

// requestCaps sends CAP REQ messages for all requested caps
func (cm *CapabilityManager) requestCaps() error {
    capsToReq := cm.filterCaps(func(capability *Capability) bool {
        return capability.requested && capability.available
    })

    cm.bot.Log.Infof("requesting capabilities: %v", capsToReq)

    var caps []string
    for _, c := range capsToReq {
        caps = append(caps, c.Name)
    }
    err := cm.bot.WriteLine(
        util.MakeSimpleIRCLine("CAP", "REQ", strings.Join(caps, " ")),
    )
    if err != nil {
        return err
    }
    return nil
}

// parseCapStr takes a cap string and splits it into the cap name and the arguments to that cap
func parseCapStr(capStr string) (string, string) {
    split := strings.SplitN(capStr, "=", 1)
    if len(split) < 2 {
        split = append(split, "")
    }
    return split[0], split[1]
}
