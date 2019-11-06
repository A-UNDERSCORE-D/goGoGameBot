package interfaces

// Bot represents a bot, that is, a connection to a chat service that we are bridging though
type Bot interface {
	Connect() error          // Connect connects to the service. It MUST NOT block after negotiation with the service is complete
	Disconnect(string)       // Disconnect disconnects the bot from the service accepts a message for a reason, and should make Run return
	Run() error              // Run runs the bot, it should Connect() to the service and MUST block until connection is lost
	SendAdminMessage(string) // Send a message to the administration channel
	Messager
	Hooker
	AdminLeveler
	JoinChannel(string)                       // Join the given channel
	Reload(string) error                      // Reload using the passed config
	CommandPrefixes() []string                // Return the current valid command prefixes
	HumanReadableSource(source string) string // Convert the provided source string to a version that is human readable
	Statuser
}

// Messager represents a type that can send messages to a chat system. Implementations should expect and handle
// newlines if needed
// TODO: This should state / expect that conversion	from format/transformer intermediate to a protocol level formatting
//       should be done with these methods
type Messager interface {
	SendMessage(target, message string)
	SendNotice(target, message string)
}

// Hooker provides methods for hooking on specific "chat" events, like joining and leaving a channel, or messages in
// a given channel
type Hooker interface {
	HookMessage(func(source, channel, message string))        // Standard chat message
	HookPrivateMessage(func(source, channel, message string)) // Private message to the bot
	HookJoin(func(source, channel string))                    // User joining a channel
	HookPart(func(source, channel, message string))           // User leaving a channel
	HookQuit(func(source, message string))                    // User disconnecting
	HookKick(func(source, channel, target, message string))   // User kicked from a channel
	HookNick(func(source, newNick string))                    // User changing their nickname
	// TODO: bans
	// TODO: system notices (this is where modes etc will go)
}

// CommandResponder provides helper methods for responding to command calls with Messages, and Notices
type CommandResponder interface {
	ReturnNotice(msg string)  // Return a message in a way that should not be responded to (and may be private)
	ReturnMessage(msg string) // Return a message that will be publicly shown in the calling channel
}

// AdminLeveler provides a method to check what admin level a given string has
type AdminLeveler interface {
	AdminLevel(source string) int // The level that the given source should be able to access
}
