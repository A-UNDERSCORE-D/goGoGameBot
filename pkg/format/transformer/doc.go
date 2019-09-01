// Package transformer implements a platform agnostic way of converting between platform specific formatting styles
//
// In order to do this, an intermediate format is used internally and by all implementations of Transformer
// The format is defined as follows
//
// Formatters are first indicated with a sentinel, '$', after which a single character for type follows
// The only defined types are:
//    b	Bold
//    i	Italics
//    u	Underline
//    s	Strikethrough
//    r	Reset
//    $	Escaped Sentinel
//    c Colour
//
// Colour has a special definition, instead of simply $c, the sentinel and type is followed by six hex characters that
// together indicate a colour. It is expected that converters with a smaller colour space automatically condense any
// unsupported colours into something within their colour space.
//
// Unintended uses of the sentinel must be escaped by using two sentinels in a row.
package transformer
