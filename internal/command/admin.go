package command

import "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"

type Admin struct {
	Level int
	Mask  string
}

func (a *Admin) MatchesMask(mask string) bool {
	return util.GlobToRegexp(a.Mask).MatchString(mask)
}
