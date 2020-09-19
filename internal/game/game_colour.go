package game

import (
	"fmt"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config/tomlconf"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer"
)

func (g *Game) setupTransformer(conf *tomlconf.Game) error {

	if conf.Chat.Transformer == nil {
		conf.Chat.Transformer = &tomlconf.ConfigHolder{Type: "strip"}
	}

	t, err := transformer.GetTransformer(conf.Chat.Transformer)
	if err != nil {
		return fmt.Errorf("could not add transformer to game %s: %w", conf.Name, err)
	}

	g.chatBridge.transformer = t

	return nil
}
