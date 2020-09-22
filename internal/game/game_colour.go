package game

import (
	"fmt"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/pkg/format/transformer"
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
