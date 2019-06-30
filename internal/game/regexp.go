package game

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"text/template"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

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

func NewRegexp(conf config.Regexp, manager *RegexpManager, root *template.Template) (*Regexp, error) {
	compiledRe, err := regexp.Compile(conf.Regexp)
	if err != nil {
		return nil, fmt.Errorf("could not compile regexp %s for mannager %s: %s", conf.Name, manager, err)
	}

	funcs := template.FuncMap{
		"sendToMsgChan":   manager.game.templSendToMsgChan,
		"sendToAdminChan": manager.game.templSendToAdminChan,
		"sendPrivmsg":     manager.game.templSendPrivmsg,
	}
	var templ *util.Format = nil
	if err := conf.Format.Compile("regexp_"+conf.Name, true, root, funcs); err != nil {
		if err != util.ErrEmptyFormat {
			return nil, fmt.Errorf("could not compile format for regexp %s on %s: %s", conf.Name, manager, err)
		}
	} else {
		templ = &conf.Format
	}

	return &Regexp{
		priority:         conf.Priority,
		regexp:           compiledRe,
		template:         templ,
		manager:          manager,
		eat:              !conf.DontEat,
		sendToChan:       !conf.DontSend,
		sendToOtherGames: !conf.DontForward,
	}, nil

}

type Regexp struct {
	priority int
	regexp   *regexp.Regexp
	template *util.Format
	manager  *RegexpManager

	eat              bool
	sendToChan       bool
	sendToOtherGames bool
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
		r.manager.game.sendToMsgChan(resp)
	}

	if r.sendToOtherGames {
		r.manager.game.writeToAllOthers(resp)
	}
	return true, nil
}

func NewRegexpManager(game *Game) *RegexpManager {
	return &RegexpManager{game: game}
}

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

func (r *RegexpManager) UpdateFromConf(res []config.Regexp, root *template.Template) error {
	var reList RegexpList
	r.game.Debug("regex manager reloading")
	for _, reConf := range res {
		re, err := NewRegexp(reConf, r, root)
		if err != nil {
			return err
		}
		r.game.Debugf("adding regexp %#v", re)
		reList = append(reList, re)
	}
	sort.Sort(reList)
	r.Lock()
	r.regexps = reList
	r.Unlock()
	r.game.Debug("regexp manager reload complete")
	return nil
}
