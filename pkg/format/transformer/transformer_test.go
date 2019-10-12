package transformer

import (
	"image/color"
)

func cmpSliceNoOrder(s1, s2 []color.Color) bool {
	if len(s1) != len(s2) {
		return false
	}
	seen := make(map[color.Color]int)
	for i, v := range s1 {
		seen[v]++
		seen[s2[i]]++
	}

	for _, v := range seen {
		if v%2 != 0 {
			return false
		}
	}
	return true
}
