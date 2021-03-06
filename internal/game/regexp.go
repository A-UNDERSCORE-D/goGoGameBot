package game

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"text/template"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/pkg/format"
)

// RegexpList is a slice of pointers to regexps that exists simply to implement the sort interface
type RegexpList []*Regexp

func (gl RegexpList) Len() int {
	return len(gl)
}

func (gl RegexpList) Less(i, j int) bool {
	return gl[i].priority < gl[j].priority
}

func (gl RegexpList) Swap(i, j int) {
	gl[i], gl[j] = gl[j], gl[i]
}

// NewRegexp instantiates a regexp object from a config.Regexp. the root may be nil, otherwise everything passed must
// exist
func NewRegexp(conf tomlconf.Regexp, manager *RegexpManager, root *template.Template) (*Regexp, error) {
	compiledRe, err := regexp.Compile(conf.Regexp)
	if err != nil {
		return nil, fmt.Errorf("could not compile regexp %s for manager %s: %s", conf.Name, manager, err)
	}

	// TODO: move this to the DataForFmt object, functions get weird when they're attached like this
	funcs := template.FuncMap{
		"sendToMsgChan": manager.game.templSendToMsgChan,
		"sendPrivmsg":   manager.game.templSendMessage, // TODO: rename this
	}

	templ := &format.Format{FormatString: conf.Format}

	if err := templ.Compile("regexp_"+conf.Name, root, funcs); err != nil {
		if err != format.ErrEmptyFormat {
			return nil, fmt.Errorf("could not compile format for regexp %s on %s: %s", conf.Name, manager, err)
		}

		templ = nil // it was empty. Means this regexp is probably used to eat a line
	}

	return &Regexp{
		priority:         conf.Priority,
		regexp:           compiledRe,
		template:         templ,
		manager:          manager,
		eat:              conf.Eat,
		sendToChan:       conf.SendToChan,
		sendToOtherGames: conf.SendToOthers,
		sendToLocalGame:  conf.SendToLocal,
	}, nil
}

// Regexp is a representation of a regex and a util.Format pair that is applied to stdout lines of a game
type Regexp struct {
	priority int
	regexp   *regexp.Regexp
	template *format.Format
	manager  *RegexpManager

	eat              bool
	sendToChan       bool
	sendToOtherGames bool
	sendToLocalGame  bool
}

func (r *Regexp) String() string {
	return fmt.Sprintf("%q with formatter %s", r.regexp, r.template.CompiledFormat.Name())
}

func (r *Regexp) matchToMap(line string) (map[string]string, bool) {
	out := make(map[string]string)

	match := r.regexp.FindStringSubmatch(line)
	if match == nil {
		return nil, false
	}

	for i, name := range r.regexp.SubexpNames() {
		if i == 0 {
			continue
		}

		if name != "" {
			out[name] = match[i]
		} else {
			out[strconv.Itoa(i)] = match[i]
		}
	}

	return out, true
}

func (r *Regexp) checkAndExecute(line string, stdout bool) (bool, error) {
	matchMap, ok := r.matchToMap(line)
	if !ok {
		return false, nil
	}

	if r.template == nil {
		// we matched, but dont have any template to use.
		// we're probably being used to strip out data
		return true, nil
	}

	data := struct {
		IsStdout bool
		Groups   map[string]string
	}{stdout, matchMap}

	resp, err := r.template.Execute(data)
	if err != nil {
		return false, fmt.Errorf(
			"cannot run game template for %s (%q): %s",
			r.manager,
			r.template.CompiledFormat.Name(),
			err,
		)
	}

	if r.sendToChan {
		r.manager.game.sendToBridgedChannel(resp)
	}

	if r.sendToOtherGames {
		r.manager.game.writeToAllOthers(resp)
	}

	if r.sendToLocalGame {
		r.manager.game.SendLineFromOtherGame(resp, r.manager.game)
	}

	return true, nil
}

// NewRegexpManager creates a new RegexpManager with reference to the passed Game
func NewRegexpManager(game *Game) *RegexpManager {
	return &RegexpManager{game: game}
}

// RegexpManager manages regexps for a game
type RegexpManager struct {
	sync.RWMutex
	regexps RegexpList
	game    *Game
}

func (r *RegexpManager) checkAndExecute(line string, isStdout bool) {
	r.RLock()
	defer r.RUnlock()

	for _, reg := range r.regexps {
		if matched, err := reg.checkAndExecute(line, isStdout); err != nil {
			r.game.manager.Error(err)
			continue
		} else if matched && reg.eat {
			break
		}
	}
}

func (r *RegexpManager) String() string {
	return fmt.Sprintf("game.RegexpManager at %p attached to %s", r, r.game)
}

// UpdateFromConf updates the regexps on the RegexpManager. During the update, the Regexp list is sorted. UpdateFromConf
// only applies changes if none of the regexps failed to be created with NewRegexp. It is safe for concurrent use
func (r *RegexpManager) UpdateFromConf(res []tomlconf.Regexp, root *template.Template) error {
	var reList RegexpList

	r.game.Debug("regex manager reloading")

	for _, reConf := range res {
		re, err := NewRegexp(reConf, r, root)
		if err != nil {
			return err
		}

		r.game.Debugf("adding regexp %s", re)
		reList = append(reList, re)
	}

	sort.Sort(reList)
	r.Lock()
	r.regexps = reList
	r.Unlock()
	r.game.Debug("regexp manager reload complete")

	return nil
}
