package interfaces

// Bot represents a bot
type Bot interface {
	Connect() error
	Disconnect(string)
	Run() error
	SendAdminMessage(string)
	Messager
	Hooker
	AdminLeveler
	JoinChannel(string)
	Reload(string) error
	CommandPrefixes() []string
	HumanReadableSource(source string) string
}

// Messager represents a type that can send messages to a chat system
// TODO: This should state / expect that conversion	from format/transformer intermediate to a protocol level formatting
//       should be done with these methods
type Messager interface {
	SendMessage(target, message string)
	SendNotice(target, message string)
}

// Hooker provides methods for hooking on specific "chat" events, like joining and leaving a channel, or messages in
// a given channel
type Hooker interface {
	HookMessage(func(source, channel, message string))
	HookPrivateMessage(func(source, channel, message string))
	HookJoin(func(source, channel string))
	HookPart(func(source, channel, message string))
	HookQuit(func(source, message string))
	HookKick(func(source, channel, target, message string))
	HookNick(func(source, newNick string))
	// TODO: bans
	// TODO: system notices (this is where modes etc will go)
}

// CommandResponder provides helper methods for responding to command calls with Messages, and Notices
type CommandResponder interface {
	ReturnNotice(msg string)
	ReturnMessage(msg string)
}

// AdminLeveler provides a method to check what admin level a given string has
type AdminLeveler interface {
	AdminLevel(source string) int
}
