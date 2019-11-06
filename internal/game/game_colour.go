package game

import (
	"fmt"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer"
)

func (g *Game) setupTransformer(game config.Game) error {
	if game.Chat.TransformerConfig.Type == "" {
		game.Chat.TransformerConfig.Type = "strip"
	}

	t, err := transformer.GetTransformer(game.Chat.TransformerConfig)
	if err != nil {
		return fmt.Errorf("could not add transformer to game %s: %w", game.Name, err)
	}
	g.chatBridge.transformer = t
	return nil
}
