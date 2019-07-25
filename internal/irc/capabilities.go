package irc

import (
	"strings"
	"sync"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

type capability struct {
	name      string
	args      string
	available bool // Has the server said it exists
	requested bool // Have we asked the server for it
	enabled   bool // Has the server ACK-ed our REQ?
	supported bool // Do we locally support this?
}

type capabilityManager struct {
	sync.RWMutex
	caps    []*capability
	irc     *IRC
	capChan chan *RawEvent
	counter int

	// for use during negotiation
	doingInitialNegotiation bool
	waitingForMoreCaps      bool
	offeredCaps             []string
	ackedCaps               []string
}

func newCapabilityManager(irc *IRC) *capabilityManager {
	return &capabilityManager{irc: irc, capChan: make(chan *RawEvent)}
}

func (c *capabilityManager) supportCap(name string) {
	c.addOrGetCap(name).supported = true
}

func (c *capabilityManager) getCapByName(name string) *capability {
	name = strings.SplitN(name, "=", 2)[0]
	c.RLock()
	defer c.RUnlock()
	for _, capability := range c.caps {
		if capability.name == name {
			return capability
		}
	}

	return nil
}

func (c *capabilityManager) onLine(e event.Event) {
	if raw := event2RawEvent(e); raw != nil {
		c.capChan <- raw
	}
}

func (c *capabilityManager) negotiateCaps() {
	c.doingInitialNegotiation = true
	capID := c.irc.RawEvents.Attach("CAP", c.onLine, event.PriHighest)
	welcomeID := c.irc.RawEvents.AttachOneShot("001", c.onLine, event.PriHighest)
	defer c.irc.RawEvents.Detach(capID)
	defer c.irc.RawEvents.Detach(welcomeID)

	_, err := c.irc.writeLine("CAP", "LS", "302")
	if err != nil {
		c.irc.log.Warn("could not write CAP LS command. Capability negotiation aborted")
		return
	}

	for c.doingInitialNegotiation {
		ev := <-c.capChan
		if ev.CommandIs("001") {
			c.irc.log.Warn("got an unexpected 001 while waitingForMoreCaps on capabilities. Assuming the server does not support caps and aborting negotiation")
			return
		}

		args := ev.Line.Params
		switch args[1] {
		case "LS":
			c.handleLS(args)
		case "ACK":
			c.handleACK(args)
		case "NAK":
			c.handleNAK(args)
		case "DEL":
			c.handleDEL(args)
		case "NEW":
			c.handleNEW(args)
		}
	}
	c.irc.writeLine("CAP", "END")
}

func (c *capabilityManager) handleLS(args []string) {
	c.waitingForMoreCaps = util.ReverseIdx(args, -2) == "*"
	c.offeredCaps = append(c.offeredCaps, strings.Split(util.ReverseIdx(args, -1), " ")...)

	if !c.waitingForMoreCaps {
		c.irc.log.Info("server offered capabilities: ", strings.Join(c.offeredCaps, ", "))
		for _, name := range c.offeredCaps {
			c.addOrGetCap(name).available = true
		}
		c.requestCaps()
	}
}

func (c *capabilityManager) handleACK(args []string) {
	// we can reuse this here because we should never get an ACK during an LS
	c.waitingForMoreCaps = util.ReverseIdx(args, -2) == "*"
	c.ackedCaps = append(c.ackedCaps, strings.Split(util.ReverseIdx(args, -1), " ")...)

	if !c.waitingForMoreCaps {
		c.irc.log.Info("server accepted capabilities: ", strings.Join(c.ackedCaps, ", "))
		for _, name := range c.ackedCaps {
			capab := c.getCapByName(name)
			if capab == nil {
				c.irc.log.Warn("Server acknowledged a capability we dont know. Ignoring")
				return
			}

			capab.enabled = true
		}
		c.doingInitialNegotiation = false
	}
}

func (c *capabilityManager) handleNAK(args []string) {
	caps := strings.Split(util.ReverseIdx(args, -1), " ")
	c.irc.log.Warn("server did not acknowledge some capabilities we asked for: ", strings.Join(caps, ", "))
	for _, capabName := range caps {
		if capab := c.getCapByName(capabName); capab != nil {
			capab.enabled = false
		} else {
			c.irc.log.Warn("server NAK-ed a capability we dont have: ", capabName)
		}
	}
}

func (c *capabilityManager) handleDEL(args []string) {
	caps := strings.Split(util.ReverseIdx(args, -1), " ")
	c.irc.log.Info("server disabled capabilities: ", strings.Join(caps, ", "))
	for _, capName := range caps {
		if capab := c.getCapByName(capName); capab != nil {
			capab.enabled = false
		} else {
			c.irc.log.Warn("server disabled a capability we dont have: ", capName)
		}
	}
}

func (c *capabilityManager) handleNEW(args []string) {
	caps := strings.Split(util.ReverseIdx(args, -1), " ")
	c.irc.log.Info("server added new capabilities: ", strings.Join(caps, ", "))
	for _, capName := range caps {
		capab := c.addOrGetCap(capName)
		if capab.supported {
			capab.available = true
		}
	}
	// we have new caps, are any of them ones we want?
	c.requestCaps()
}

func (c *capabilityManager) addOrGetCap(name string) *capability {
	var args = ""
	if strings.Contains(name, "=") {
		split := strings.SplitN(name, "=", 2)
		name = split[0]
		args = util.IdxOrEmpty(split, 1)
	}

	if existing := c.getCapByName(name); existing != nil {
		return existing
	}

	toAdd := &capability{name: name, args: args}
	c.Lock()
	c.caps = append(c.caps, toAdd)
	c.Unlock()
	return toAdd
}

func (c *capabilityManager) requestCaps() {
	c.Lock()
	defer c.Unlock()
	var toReq []string
	for _, capab := range c.caps {
		if capab.enabled || !capab.available || !capab.supported {
			continue
		}
		toReq = append(toReq, capab.name)
		capab.requested = true
	}
	if len(toReq) == 0 {
		c.irc.log.Info("no capabilities to request, ending negotiation")
		c.doingInitialNegotiation = false
		return
	}

	c.irc.log.Info("requesting capabilities: ", strings.Join(toReq, ", "))
	for _, capSet := range util.JoinToMaxLength(toReq, " ", 50) {
		c.irc.writeLine("CAP", "REQ", capSet)
	}
}
