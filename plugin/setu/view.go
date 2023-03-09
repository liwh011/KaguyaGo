package setu

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/liwh011/gonebot"
	gbmw "github.com/liwh011/gonebot/middlewares"
)

var config = struct {
	cd          int // 同一用户冲的间隔，秒
	times       int // 同一用户每天可以冲的次数，秒
	deleteDelay int // 图片发送后撤回的延迟，秒，应小于2分钟
}{
	cd:          5,
	times:       5,
	deleteDelay: 30,
}

func init() {
	gonebot.RegisterPlugin(&SetuPlugin{}, &config)
}

type SetuPlugin struct{}

func (SetuPlugin) GetPluginInfo() gonebot.PluginInfo {
	return gonebot.PluginInfo{
		Name:        "setu",
		Description: "setu",
		Author:      "011",
		Version:     "0.0.1",
	}
}

func (SetuPlugin) Init(engine *gonebot.PluginHub) {
	// 用户在不同群的数据都应该是共通的
	keyFunc := func(ctx *gonebot.Context) string {
		return fmt.Sprintf("%d", ctx.Event.(*gonebot.GroupMessageEvent).UserId)
	}

	freqLimiter := gbmw.NewFrequencyLimiter(config.cd, keyFunc)
	freqLimiter.OnFail(func(ctx *gonebot.Context) {
		ctx.Reply("你冲得太快了，休息一下吧")
	})

	timesLimiter := gbmw.NewDailyTimesLimiter(config.times, keyFunc)
	timesLimiter.OnFail(func(ctx *gonebot.Context) {
		ctx.Replyf("您今天已经冲过%d次了，请%s后再来！", timesLimiter.Times, timesLimiter.GetResetTime().Format("02日15:04"))
	})

	engine.NewHandler(gonebot.EventName_GroupMessage).
		Use(gonebot.Regex(*regexp.MustCompile("^来点(.*?)的[涩瑟色]图"))).
		Use(freqLimiter.Handle).
		Use(timesLimiter.Handle).
		Handle(func(ctx *gonebot.Context) {
			raw := ctx.GetRegexMatchResult().Get(1)
			// raw := ctx.GetMap("regex")["matched"].([]string)[1]
			tags := regexp.MustCompile("[,， ]+").Split(raw, -1)
			pics, err := FetchOnlinWithTags(tags)
			if err != nil {
				ctx.Replyf("获取图片错误: %s", err)
				return
			}
			if len(pics) == 0 {
				ctx.Reply("没有符合的图片，你的xp好怪喔")
				return
			}

			sendPic(ctx, pics)
		})

	engine.NewHandler(gonebot.EventName_GroupMessage).
		Use(gonebot.Regex(*regexp.MustCompile("^不够[涩瑟色]|[涩瑟色]图|来一?[点份张].*[涩瑟色]|再来[点份张]|看过了|铜$"))).
		Use(timesLimiter.Handle).
		Use(freqLimiter.Handle).
		Handle(func(ctx *gonebot.Context) {
			pics, err := FetchOnlineRandom()
			if err != nil {
				ctx.Replyf("获取图片错误: %s", err)
				return
			}
			if len(pics) == 0 {
				ctx.Reply("找不到图片，也许发生什么错误了")
				return
			}

			sendPic(ctx, pics)
		})
}

func sendPic(ctx *gonebot.Context, pics []Pic) {
	pic := pics[rand.Intn(len(pics))]
	msg, _ := gonebot.MsgPrintf(
		"{}\n标题: %s\nPID: %d\n作者: %s\n",
		gonebot.MsgFactory.Image(pic.Urls.Original, nil),
		pic.Title,
		pic.Pid,
		pic.Author,
	)

	gid := ctx.Event.(*gonebot.GroupMessageEvent).GroupId
	msgId, err := ctx.Bot.SendGroupMsg(gid, msg, false)
	if err != nil {
		ctx.Replyf("图片发不出去了...%s", err)
	}
	// 发送成功后过一会撤回
	if msgId != 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(config.deleteDelay))
			ctx.Bot.DeleteMsg(msgId)
		}()
	}
}
