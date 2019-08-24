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
