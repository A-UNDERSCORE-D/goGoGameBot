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
type GameRegexp struct {
    Name      string
    watcher   watchers.Watcher
    template  *template.Template
    Priority  int
    game      *Game
    shouldEat bool
}

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
    if c.Priority != -1  {
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
