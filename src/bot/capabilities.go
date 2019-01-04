package bot

import (
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util"
    "github.com/goshuirc/irc-go/ircmsg"
    "strings"
    "sync"
)

type Capability struct {
    Name      string
    Params    string
    Callback  func(*Capability, ircmsg.IrcMessage, *sync.WaitGroup)
    requested bool
    available bool
    enabled   bool
}

func (c *Capability) String() string {
    return fmt.Sprintf("%s: %s", c.Name, c.Params)
}

type CapabilityManager struct {
    capabilities []*Capability
    bot          *Bot
}

func (cm *CapabilityManager) requestCap(capability *Capability) {
    capability.requested = true
    cm.capabilities = append(cm.capabilities, capability)
}

func (cm *CapabilityManager) getCapByName(name string) *Capability {
    for _, ca := range cm.capabilities {
        if ca.Name == name {
            return ca
        }
    }
    return nil
}

func (cm *CapabilityManager) hasCap(requestedCap string) bool {
    if c := cm.getCapByName(requestedCap); c != nil {
        return c.available
    }
    return false
}

func (cm *CapabilityManager) filterCaps(f func(*Capability) bool) []*Capability {
    var out []*Capability
    for _, c := range cm.capabilities {
        if f(c) {
            out = append(out, c)
        }
    }
    return out
}

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
    wg.Add(1)
    // TODO(A_D): This is an ugly as hell way of doing this. there has to be a better way
    firstRun := true
    go func() {
        // Allowing other things to run, then close up and send cap end
        wg.Wait()
        close(done)
        cm.bot.WriteLine(util.MakeSimpleIRCLine("CAP", "END"))
    }()

    for line := range lineChan {
        switch line.Params[1] {
        case "LS":
            cm.addAvailableCapability(line)
            if !firstRun {
                wg.Add(1)
            } else {
                firstRun = false
            }
            if line.Params[2] == "*" {
                continue // This is multiline, lets just append for now
            }

            cm.bot.Log.Printf("Server offered caps: %v",
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
                    cm.bot.Log.Printf(
                        "[WARN] Got an ACK for a cap we dont know about: %q with params %q", name, params,
                    )
                }
            }
            wg.Done()

        case "NAK":
            // To be implemented >.>
            panic("Not Implemented")
        }
    }
}

func (cm *CapabilityManager) addAvailableCapability(line ircmsg.IrcMessage) {
    for _, capStr := range strings.Split(line.Params[len(line.Params) - 1], " ") {
        name, params := parseCapStr(capStr)
        if c := cm.getCapByName(name); c != nil {
            c.available = true
            c.Params = params
            cm.bot.Log.Printf("%#v", c)
        } else {
            cm.capabilities = append(
                cm.capabilities,
                &Capability{Name: name, Params: params, requested: false, available: true},
            )
        }
    }
}

func (cm *CapabilityManager) requestCaps() error {
    capsToReq := cm.filterCaps(func(capability *Capability) bool {
        return capability.requested && capability.available
    })

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

func parseCapStr(capStr string) (string, string) {
    split := strings.SplitN(capStr, "=", 1)
    if len(split) < 2 {
        split = append(split, "")
    }
    return split[0], split[1]
}
