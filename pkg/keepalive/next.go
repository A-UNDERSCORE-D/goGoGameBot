// Package keepalive contains various strings to use for keepalives.
//
// its essentially a bunch of references I added for fun, but it does serve the purpose of providing changing data
// for use in keepalives
package keepalive

var curSet = 0
var sets = [][]string{
	words,
	stayinAlive,
	b5PilotMonologe,
	b5MonologueS1,
	b5MonologueS2,
	b5S2FinalNotes,
	b5MonologueS3,
	b5S3FinalNotes,
	b5MonologueS4,
	b5S4FinalNotes,
}
var curIdx = 0

// TODO: make this have a struct that stores the numbers, rather than doing it with globals. that way multiple things
// TODO: can have their own response sets
// Next returns the next string in the set, wrapping around if needed
func Next() string {
	if curIdx > len(sets[curSet])-1 {
		curIdx = 0

		curSet++
		if curSet > len(sets)-1 {
			curSet = 0
		}
	}

	out := sets[curSet][curIdx]
	curIdx++

	return out
}
