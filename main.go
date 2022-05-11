package main

import (
	_ "bot/plugin"

	"github.com/liwh011/gonebot"
)

func main() {
	cfg := gonebot.LoadConfig("./config.yaml")
	engine := gonebot.NewEngine(cfg)
	engine.Run()
}
