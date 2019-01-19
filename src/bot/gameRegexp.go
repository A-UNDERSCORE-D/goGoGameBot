package bot

import (
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util/watchers"
    "strings"
    "text/template"
)

const (
    DEFAULTPRIORITY = 50
)

// TODO: Set up the template with funcs to send to channels etc. That way it can be used to send out to channels etc.
//       string returned will probably be unused or logged to stdout

// TODO: adding to the above. add a target channel that the message by default goes to, if unset it goes to the chat channel
//       along with this it needs a silent bool to let the person configuring it do some... macro? programming in the middle
//       of execution

// GameRegexp is a representation of a matcher for the stdout of a process, and a text/template to apply to that line
// if it matches.
type GameRegexp struct {
    Name      string
    watcher   watchers.Watcher
    template  *template.Template
    Priority  int
    game      *Game
    shouldEat bool
}

type GameRegexpList []*GameRegexp

func (gl GameRegexpList) Len() int {
    return len(gl)
}

func (gl GameRegexpList) Less(i, j int) bool {
    return gl[i].Priority < gl[j].Priority
}

func (gl GameRegexpList) Swap(i, j int) {
    gl[i], gl[j] = gl[j], gl[i]
}

// NewGameRegexp creates a gameRegexp, compiling the relevant data structures as needed
func NewGameRegexp(game *Game, c config.GameRegexp) (*GameRegexp, error) {
    w, err := watchers.NewRegexWatcher(c.Regexp)
    if err != nil {
        return nil, err
    }

    t, err := template.New(c.Name).Funcs(nil).Parse(c.Format)
    if err != nil {
        return nil, err
    }

    p := DEFAULTPRIORITY
    if c.Priority != -1 {
        p = c.Priority
    }

    return &GameRegexp{
        game:     game,
        Name:     c.Name,
        watcher:  w,
        template: t,
        Priority: p,
    }, nil
}

// CheckAndExecute checks the given line against the stored regexp, if it matches the given template is run, and the
// result is returned
func (g *GameRegexp) CheckAndExecute(line string, stderr bool) (bool, string, error) {
    isMatched, match := g.watcher.MatchLine(line)
    if !isMatched {
        return false, "", nil
    }
    out := new(strings.Builder)
    if err := g.template.Execute(out, match); err != nil {
        g.game.bot.Error(fmt.Errorf("could not run game template %q for %q: %s", g.game.Name, g.Name, err))
        return false, "", nil
    }

    return g.shouldEat, out.String(), nil
}

func (g *GameRegexp) String() string {
    return fmt.Sprintf("GameRegexp(%s)", g.Name)
}
