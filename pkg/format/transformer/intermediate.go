package transformer

// All of the formats specified above are available here. It is expected that implementations use this wherever
// possible to allow for changes
const (
	Sentinel      = '$'
	Bold          = 'b'
	Italic        = 'i'
	Underline     = 'u'
	Strikethrough = 's'
	Reset         = 'r'
	Colour        = 'c'
)

// String representations of intermediate runes
const (
	SentinelString      = string(Sentinel)
	BoldString          = string(Bold)
	ItalicString        = string(Italic)
	UnderlineString     = string(Underline)
	StrikethroughString = string(Strikethrough)
	ResetString         = string(Reset)
	ColourString        = string(Colour)
)

// String representations of intermediate runes with a prefixed sentinel
const (
	SSentinelString      = SentinelString + SentinelString
	SBoldString          = SentinelString + BoldString
	SItalicString        = SentinelString + ItalicString
	SUnderlineString     = SentinelString + UnderlineString
	SStrikethroughString = SentinelString + StrikethroughString
	SResetString         = SentinelString + ResetString
	SColourString        = SentinelString + ColourString
)
