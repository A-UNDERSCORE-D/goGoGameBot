package watchers

type Watcher interface {
	MatchLine(string) (bool, MatchedLine)
}

type MatchedLine struct {
	Groups     map[string]string
	OutputType string
}
