package game

import "awesome-dragon.science/go/goGoGameBot/internal/interfaces"

var _ interfaces.Game = &Game{} // Make sure that Game is actually satisfying that interface
