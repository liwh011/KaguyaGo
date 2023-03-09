package pluginmanage_test

import (
	"testing"
	"time"

	"github.com/liwh011/gonebot"
	"github.com/liwh011/gonebot/mock"
	mockProvider "github.com/liwh011/gonebot/providers/mock"
)

var mockServerOptions mock.NewMockServerOptions = mock.NewMockServerOptions{
	BotId: 114514,
	Friends: []mock.User{
		{
			UserId:   1919810,
			Nickname: "至高无上的SU",
		},
	},
	Groups: []mock.Group{
		{
			GroupId:   90000001,
			GroupName: "测试群",
			Members:   []mock.GroupMember{},
		},
	},
}

var config gonebot.BaseConfig = gonebot.BaseConfig{
	ApiCallTimeout: 10,
	Superuser:      []int64{1919810},
}

func Test_privateSession(t *testing.T) {
	server := mock.NewMockServer(mockServerOptions)
	provider := mockProvider.NewMockProvider(server)
	engine := gonebot.NewEngineWithProvider(&config, provider)
	go engine.Run()

	time.Sleep(time.Millisecond * 200)
	server.ConnectedEvent()
	session := server.NewPrivateSession(1919810)
	session.MessageEventByText("plugin ls -p plugin_manage@liwh011")
	time.Sleep(time.Millisecond * 200)
	t.Log(session.GetMessageHistory())
}
