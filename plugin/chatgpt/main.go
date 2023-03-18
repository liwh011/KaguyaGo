package chatgpt

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liwh011/gonebot"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

var config = struct {
	Rpm       int
	SecretKey string
	Proxy     string
}{
	Rpm: 20,
}

func init() {
	gonebot.RegisterPlugin(&ChatgptPlugin{}, &config)
}

type RpmCounter struct {
	requestCount    int
	periodStartTime time.Time
	rpm             int
}

func (r *RpmCounter) Require() bool {
	if time.Since(r.periodStartTime) > time.Minute {
		r.requestCount = 0
		r.periodStartTime = time.Now()
	}
	r.requestCount++
	return r.requestCount <= r.rpm
}

type ChatgptPlugin struct {
	rpmCounter     RpmCounter
	sessionManager sessionManager
	chatgptClient  *chatgpt
	storage        gonebot.Storage
	lock           sync.Mutex
}

func (plugin *ChatgptPlugin) GetPluginInfo() gonebot.PluginInfo {
	return gonebot.PluginInfo{
		Name:        "chatgpt",
		Description: "chatgpt on gonebot",
		Author:      "liwh011",
		Version:     "0.0.1",
	}
}

var usage string = `
1. 列出群中的历史会话
使用命令："cgpt session ls [-s <会话ID>] [-p <页码>]"
作用：列出当前群聊中的历史会话列表。
可选参数："-s" 要详细展示的会话，省略则查看全部会话；"-p" 要查看的页码，省略则表示第一页

2. 新建会话，使用新的催眠
使用命令："cgpt session new [-n <会话名称>] [prompt]"
作用：在当前群聊中新建一个会话，并启用一个新的催眠。
可选参数："prompt" 要使用的催眠咒语，省略则使用默认咒语(猫娘)；"-n" 新会话的名称，省略则使用默认名称

3. 清空当前会话，但保留催眠咒语
使用命令："cgpt session reset"
作用：使用当前催眠咒语创建新会话，旧会话依旧存在。

4. 切换会话
使用命令："cgpt session switch <会话ID>"
作用：在当前群聊中切换不同的会话。

5. 重命名会话
使用命令："cgpt session rename [-s <会话ID>] <新名称>"
作用：为指定会话设置新的名称。
可选参数："-s" 会话id或名称，省略则表示重命名当前会话
`

func (plugin *ChatgptPlugin) Init(engine *gonebot.PluginHub) {
	if config.SecretKey == "" {
		logrus.Fatal("Chatgpt secret key not set")
	}

	storage, err := gonebot.NewStorage("chatgpt")
	if err != nil {
		logrus.Fatalf("Failed to create storage: %v", err)
	}
	plugin.storage = storage
	plugin.restoreSessions()

	plugin.rpmCounter.rpm = config.Rpm
	plugin.chatgptClient = newChatgpt(config.SecretKey, config.Proxy)

	// engine.NewHandler(gonebot.EventName_PrivateMessage).
	// 	Use(gonebot.FromSuperuser()).
	// 	Handle(func(ctx *gonebot.Context) {
	// 		ev := ctx.Event.(*gonebot.PrivateMessageEvent)

	// 	})

	engine.NewHandler(gonebot.EventName_GroupMessage).
		Use(gonebot.FromAdminOrHigher()).
		Use(gonebot.StartsWith("cgpt help")).
		Handle(func(ctx *gonebot.Context) {
			text := strings.TrimPrefix(ctx.Event.ExtractPlainText(), "cgpt help")
			text = strings.TrimSpace(text)
			logrus.Debugf("正在调用Chatgpt来回复问题：%s", text)
			rep, err := plugin.chatgptClient.ReplySingle(fmt.Sprintf("下面是cgpt命令的使用说明：%s\n请你根据使用说明来回答用户的问题并给出使用示例，帮助用户使用该命令。用户的问题为：%s", usage, text))
			if err != nil {
				ctx.Reply(fmt.Sprintf("Chatgpt错误：%v", err))
				return
			}
			ctx.Reply(rep)
		})

	engine.NewHandler(gonebot.EventName_GroupMessage).
		Use(gonebot.FromAdminOrHigher()).
		Use(gonebot.ShellLikeCommand("cgpt", &groupCmd{}, gonebot.ParseFailedAction_AutoReply)).
		Handle(func(ctx *gonebot.Context) {
			parseResult := ctx.GetShellLikeCommandResult()
			if !parseResult.HasSubcommand() {
				msg := parseResult.FormatErrorAndHelp(fmt.Errorf("未指定子命令"))
				ctx.Reply(msg)
				return
			}
			parseResult.GetSubcommand().(Command).Run(ctx, plugin)
			plugin.storeSessions()
		})

	debouncer := NewDebouncer(4 * time.Second)

	engine.NewHandler(gonebot.EventName_GroupMessage).
		Handle(func(ctx *gonebot.Context) {
			if ctx.Next() {
				return
			}

			ev := ctx.Event.(*gonebot.GroupMessageEvent)
			groupId := ev.GroupId

			chatsession := plugin.sessionManager.GetGroupCurrentSession(groupId)
			if chatsession == nil {
				chatsession = plugin.sessionManager.CreateGroupSession(groupId, "")
			}

			text := ""
			for _, m := range ev.Message {
				if m.Type == "at" {
					if m.Data["qq"] == ev.SelfId {
						continue
					}
					text += fmt.Sprintf("@%s ", m.Data["qq"])
				} else if m.Type == "text" {
					text += m.Data["text"].(string)
				}
			}
			if len(text) == 0 {
				return
			}

			toMe := ev.IsToMe()

			chatsession.AddHistory(ev.Sender.UserId, text, toMe)

			if !toMe {
				possiblity := 60
				if len(chatsession.History) < 5 {
					possiblity = 10
				}
				if rand.Intn(100) >= possiblity {
					return
				}
			}

			debouncer(func() {
				logrus.Debug("正在调用Chatgpt")
				reply, err := plugin.chatgptClient.Reply(chatsession)
				if err != nil {
					if errors.Is(err, &openai.APIError{}) {
						ctx.Reply("Chatgpt调用错误: " + err.Error())
						return
					} else {
						ctx.Replyf("请求错误: %v", err)
						return
					}
				}

				// 转换@到消息
				replyMsg := gonebot.Message{}
				splitByAt := strings.Split(reply, "@")
				replyMsg.AppendText(splitByAt[0])
				regexp := regexp.MustCompile(`([0-9]+) (.*)`)
				if len(splitByAt) > 1 {
					for _, s := range splitByAt[1:] {
						matches := regexp.FindStringSubmatch(s)
						if matches == nil {
							replyMsg.AppendText("@" + s)
							continue
						}
						userId := matches[1]
						userIdInt, _ := strconv.ParseInt(userId, 10, 64)
						otherText := matches[2]
						replyMsg.Append(gonebot.MsgFactory.AtSomeone(userIdInt))
						replyMsg.AppendText(otherText)
					}
				}

				ctx.AtSender(toMe)
				ctx.ReplyMsg(replyMsg)
				plugin.lock.Lock()
				defer plugin.lock.Unlock()
				chatsession.AddHistory(114514, reply, false)
				plugin.storeSessions()
			})

		})
}

func (plugin *ChatgptPlugin) storeSessions() {
	plugin.storage.Set("sessions", "session_manager", plugin.sessionManager)
}

func (plugin *ChatgptPlugin) restoreSessions() {
	plugin.storage.Get("sessions", "session_manager", &plugin.sessionManager)
}
