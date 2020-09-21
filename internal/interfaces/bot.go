package interfaces

// Bot represents a bot, that is, a connection to a chat service that we are bridging though
type Bot interface {
	// Connect connects to the service. It MUST NOT block after negotiation with the service is complete
	Connect() error
	// Disconnect disconnects the bot from the service accepts a message for a reason, and should make Run return
	Disconnect(msg string)
	// Run runs the bot, it should Connect() to the service and MUST block until connection is lost
	Run() error
	// SendAdminMessage sends the given message to the administrator
	SendAdminMessage(msg string)
	// SendRaw sends a raw message across the Bot
	SendRaw(raw string)
	Messager // TODO: fix this spelling
	Hooker
	AdminLeveller
	// JoinChannel joins the given channel
	JoinChannel(name string)
	// Reload reloads the Bot using the given unmarshalable config
	Reload(unmarshaller Unmarshaler) error
	// StaticCommandPrefixes returns the bots current static prefixes
	// ie, ones that will not change over the bot lifetime
	StaticCommandPrefixes() []string
	// IsCommandPrefix returns whether or not the given string starts with a command prefix
	// if it does, it also returns the given string with the prefix removed
	IsCommandPrefix(string) (string, bool)
	// HumanReadableSource converts the given message source to one that is human readable
	HumanReadableSource(source string) string
	Statuser //nolint:misspell // intentional.
}

// Messager represents a type that can send messages to a chat system. Implementations should expect and handle
// newlines if needed. Implementations should also convert incoming lines to their protocol level formatting if
// applicable
type Messager interface {
	// SendMessage sends a message to the given target
	SendMessage(target, message string)
	// SendNotice sends a message that should not be responded to to the given target
	SendNotice(target, message string)
}

// Hooker provides methods for hooking on specific "chat" events, like joining and leaving a channel, or messages in
// a given channel. It is expected that all these callbacks receive messages in the intermediate format
type Hooker interface {
	// HookMessage hooks on messages to a channel
	HookMessage(func(source, channel, message string, isAction bool))
	// HookPrivateMessage hooks on messages to us directly
	HookPrivateMessage(func(source, channel, message string))
	// HookJoin hooks on users joining a channel
	HookJoin(func(source, channel string))
	// HookPart hooks on users leaving a channel
	HookPart(func(source, channel, message string))
	// HookQuit hooks on users disconnecting
	HookQuit(func(source, message string)) // (for services that differentiate it from the above)
	// HookKick hooks on a user being kicked from a channel
	HookKick(func(source, channel, target, message string))
	// HookNick hoops on a user changing their nickname
	HookNick(func(source, newNick string))

	// TODO: bans, system notices (this is where modes etc will go)
}

// CommandResponder provides helper methods for responding to command calls with Messages, and Notices
type CommandResponder interface {
	// ReturnNotice returns a message in a way that should not be responded to
	// (and should be private)
	ReturnNotice(msg string)
	// ReturnMessage returns a message that is shown as a standard message where the command was called
	ReturnMessage(msg string)
}

// AdminLeveller provides a method to check what admin level a given string has
type AdminLeveller interface {
	// AdminLevel returns the permission level that a given source has. It should return
	// zero for sources with no permissions
	AdminLevel(source string) int
}
