package bot

import (
    "fmt"
    "text/template"

    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/watchers"
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
    Name             string
    watcher          watchers.Watcher
    template         util.Format
    Priority         int
    game             *Game
    shouldEat        bool
    shouldSendToChan bool
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
func NewGameRegexp(game *Game, c config.GameRegexpConfig) (*GameRegexp, error) {
    w, err := watchers.NewRegexWatcher(c.Regexp)
    if err != nil {
        return nil, err
    }
    funcs := template.FuncMap{
        "logchan":   game.templSendToLogChan,
        "adminchan": game.templSendToAdminChan,
        "sendto":    game.templSendPrivmsg,
    }
    game.log.Debug(funcs)

    err = c.Format.Compile(c.Name, funcs)
    if err != nil {
        return nil, err
    }

    p := DEFAULTPRIORITY
    if c.Priority != -1 {
        p = c.Priority
    }

    return &GameRegexp{
        game:             game,
        Name:             c.Name,
        watcher:          w,
        template:         c.Format,
        Priority:         p,
        shouldSendToChan: c.SendToChan,
        shouldEat:        c.ShouldEat,
    }, nil
}

// CheckAndExecute checks the given line against the stored regexp, if it matches the given template is run, and the
// result is returned
func (g *GameRegexp) CheckAndExecute(line string, stderr bool) (bool, error) {
    isMatched, match := g.watcher.MatchLine(line)
    if !isMatched {
        return false, nil
    }

    if stderr {
        match.OutputType = "STDERR"
    } else {
        match.OutputType = "STDOUT"
    }

    out, err := g.template.Execute(match)
    if err != nil {
        g.game.bot.Error(fmt.Errorf("could not run game template %q for %q: %s", g.game.Name, g.Name, err))
        return false, nil
    }

    if g.shouldSendToChan {
        g.game.sendToLogChan(out)
    }

    return g.shouldEat, nil
}

func (g *GameRegexp) String() string {
    return fmt.Sprintf("GameRegexp(%s)", g.Name)
}
