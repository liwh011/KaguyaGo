package nonebotinteract

import (
	"encoding/json"

	"github.com/liwh011/gonebot"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type Dispatcher struct {
	ws  *WebsocketServer
	bot *gonebot.Bot
}

func NewDispatcher(ws *WebsocketServer) *Dispatcher {
	d := &Dispatcher{
		ws: ws,
	}
	ws.AddHandler(d.handleRecvMsg)
	return d
}

func (d *Dispatcher) sendMsg(data interface{}, retChan chan interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	d.ws.Send(jsonData)
	return nil
}

func (d *Dispatcher) handleRecvMsg(data []byte) {
	json := gjson.ParseBytes(data)
	action := json.Get("action").String()
	params := json.Get("params").Value().(map[string]interface{})
	echo := json.Get("echo").String()

	if d.bot == nil {
		logrus.Warn("收到消息但未初始化Bot")
		return
	}
	res, err := d.bot.CallApi(action, params)
	if err != nil {
		return
	}
	resValue := res.Value().(map[string]interface{})
	resValue["echo"] = echo
	d.sendMsg(resValue, nil)
}

func (d *Dispatcher) ForwardEvent(event gonebot.I_Event) {
	d.sendMsg(event, nil)
}
