package nonebotinteract

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebsocketServer struct {
	port      int
	client    *websocket.Conn
	sendChan  chan []byte
	handler   []func(data []byte)
	onConnect func(*websocket.Conn)
}

func NewWebsocketServer(port int) *WebsocketServer {
	return &WebsocketServer{
		port: port,
	}
}

func (ws *WebsocketServer) sendMsgRoutine() {
	for {
		data := <-ws.sendChan
		err := ws.client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			logrus.WithError(err).Error("发送Websocket消息时发生错误")
		}
	}
}

func (ws *WebsocketServer) recvMsgRoutine() {
	for {
		_, data, err := ws.client.ReadMessage()
		if err != nil {
			logrus.Errorf("接收Websocket消息时发生错误：%s", err)
			return
		}
		for _, handler := range ws.handler {
			handler(data)
		}
	}
}

func (ws *WebsocketServer) Start() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.Errorf("建立Websocket连接时发生错误：%v", err)
			return
		}
		logrus.Infof("Websocket连接已建立")

		if ws.client != nil {
			ws.client.Close()
		}
		ws.client = wsConn
		ws.sendChan = make(chan []byte, 10)

		go ws.sendMsgRoutine()
		go ws.recvMsgRoutine()

		if ws.onConnect != nil {
			ws.onConnect(ws.client)
		}
	})
	logrus.Info("Websocket服务已启动")
	http.ListenAndServe(fmt.Sprintf("localhost:%d", ws.port), nil)
}

func (ws *WebsocketServer) Send(data []byte) {
	if ws.client == nil {
		return
	}
	ws.sendChan <- data
}

func (ws *WebsocketServer) AddHandler(handler func(data []byte)) {
	ws.handler = append(ws.handler, handler)
}
