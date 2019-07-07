package command

import "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"

type Admin struct {
	Level int
	Mask  string
}

// MatchesMask returns whether or not the given mask matches the mask on the Admin object
func (a *Admin) MatchesMask(mask string) bool {
	return util.GlobToRegexp(a.Mask).MatchString(mask)
}
