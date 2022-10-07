package nonebotinteract

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/liwh011/gonebot"
	"github.com/sirupsen/logrus"
)

func init() {
	gonebot.RegisterPlugin(&NonebotInteract{}, &config)
}

var config = struct {
	WebsocketPort int
}{
	WebsocketPort: 11451,
}

type NonebotInteract struct{}

func (NonebotInteract) GetPluginInfo() gonebot.PluginInfo {
	return gonebot.PluginInfo{
		Name:        "nb_interact",
		Description: "与nonebot交互",
		Author:      "liwh011",
		Version:     "0.0.1",
	}
}

func (NonebotInteract) Init(engine *gonebot.PluginHub) {
	host := NewNonebotHost(`C:\Users\liwh\AppData\Local\pypoetry\Cache\virtualenvs\adapter-HpkUz68_-py3.8\Scripts\python.exe`)
	err := host.CheckPythonVersion()
	if err != nil {
		logrus.Errorf("检查Python版本失败: %v", err)
		return
	}

	if config.WebsocketPort == -1 {
		logrus.Errorf("Websocket端口未设置，本插件将无法正常工作")
		return
	}

	ws := NewWebsocketServer(config.WebsocketPort)
	dispatcher := NewDispatcher(ws)
	ws.onConnect = func(c *websocket.Conn) {
		data := gonebot.LifeCycleMetaEvent{
			Event: gonebot.Event{
				Time:     time.Now().Unix(),
				SelfId:   engine.GetBot().GetSelfId(),
				PostType: gonebot.PostTypeMetaEvent,
			},
			MetaEventType: "lifecycle",
			SubType:       "connect",
		}
		str, _ := json.Marshal(data)
		ws.Send(str)
	}
	go ws.Start()

	go host.Start()

	engine.NewHandler().Handle(func(ctx *gonebot.Context) {
		// 不处理meta_event
		if ctx.Event.GetPostType() == gonebot.PostTypeMetaEvent {
			return
		}

		dispatcher.bot = ctx.Bot
		dispatcher.ForwardEvent(ctx.Event)
	})
}
