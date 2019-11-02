package keepalive

var curSet = 0
var sets = [][]string{words, stayinAlive}
var curIdx = 0

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
