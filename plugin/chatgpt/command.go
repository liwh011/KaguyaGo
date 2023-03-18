package chatgpt

import (
	"fmt"
	"strings"

	"github.com/liwh011/gonebot"
	"github.com/samber/lo"
)

type Command interface {
	Run(ctx *gonebot.Context, plugin *ChatgptPlugin)
}

type groupCmd struct {
	SessionManageCmd *groupSessionCmd `arg:"subcommand:session" help:"会话管理"`
}

type groupSessionCmd struct {
	ListCmd   *groupSessionListCmd   `arg:"subcommand:ls" help:"列出群中的历史会话"`
	NewCmd    *groupSessionNewCmd    `arg:"subcommand:new" help:"新建会话，使用新的催眠"`
	ResetCmd  *groupSessionResetCmd  `arg:"subcommand:reset" help:"清空当前会话，但保留催眠咒语"`
	SwitchCmd *groupSessionSwitchCmd `arg:"subcommand:switch" help:"切换会话"`
	RenameCmd *groupSessionRenameCmd `arg:"subcommand:rename" help:"重命名会话"`
}

func (cmd *groupSessionCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	res := ctx.GetShellLikeCommandResult()
	ctx.Reply(res.FormatErrorAndHelp(fmt.Errorf("未指定子命令")))
}

type groupSessionListCmd struct {
	SessionId string `arg:"-s" help:"要详细展示的会话，省略则查看全部" placeholder:"会话id或名称"`
	Page      int    `arg:"-p" help:"要查看的页码，省略则表示第一页" placeholder:"页码"`
}

func (cmd *groupSessionListCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	groupId := ctx.Event.(*gonebot.GroupMessageEvent).GroupId
	groupSessions, page, pageCount := plugin.sessionManager.ListGroupHistorySessionsByPage(groupId, cmd.Page)
	if len(groupSessions) == 0 {
		ctx.Replyf("当前群没有任何会话，发起聊天就可以建立默认会话")
		return
	}

	if cmd.SessionId == "" {
		sessionDescs := lo.Map(groupSessions, func(session *session, _ int) string {
			return session.GetBrief()
		})
		ctx.Replyf("当前群共有%d个会话，第%d/%d页为：\n%s", len(groupSessions), page, pageCount, strings.Join(sessionDescs, "\n\n"))
	} else {
		session := plugin.sessionManager.GetGroupSessionByIdOrName(groupId, cmd.SessionId)
		if session == nil {
			ctx.Replyf("会话%s不存在", cmd.SessionId)
			return
		}
		ctx.Reply(session.GetDetail())
	}
}

type groupSessionNewCmd struct {
	Prompt string `arg:"positional" help:"催眠咒语" placeholder:"催眠咒语"`
	Name   string `arg:"-n" help:"新会话的名称" placeholder:"会话名称"`
}

func (cmd *groupSessionNewCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	session := plugin.sessionManager.CreateGroupSession(ctx.Event.(*gonebot.GroupMessageEvent).GroupId, cmd.Prompt)
	if cmd.Name != "" {
		session.SessionName = cmd.Name
	}

	reply := fmt.Sprintf("已创建新会话%s", session.SessionId)
	if cmd.Name != "" {
		reply += fmt.Sprintf("，名称为%s", cmd.Name)
	} else {
		reply += "，可以通过cgpt session rename 新名称 来重命名"
	}
	ctx.Reply(reply)
}

type groupSessionResetCmd struct {
}

func (cmd *groupSessionResetCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	groupId := ctx.Event.(*gonebot.GroupMessageEvent).GroupId
	oldSession := plugin.sessionManager.GetGroupCurrentSession(groupId)
	plugin.sessionManager.ResetGroupSession(ctx.Event.(*gonebot.GroupMessageEvent).GroupId)
	ctx.Replyf("已重置当前会话，旧会话ID为：%s，可随时切换回去", oldSession.SessionId)
}

type groupSessionSwitchCmd struct {
	SessionId string `arg:"positional,required" help:"要切换到的会话id或名称" placeholder:"会话id或名称"`
}

func (cmd *groupSessionSwitchCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	groupId := ctx.Event.(*gonebot.GroupMessageEvent).GroupId
	plugin.sessionManager.SwitchGroupSessionByIdOrName(groupId, cmd.SessionId)
	ctx.Replyf("已切换会话到%s", cmd.SessionId)
}

type groupSessionRenameCmd struct {
	SessionId string `arg:"-s" help:"会话id或名称，省略则表示重命名当前会话" placeholder:"会话id或名称"`
	NewName   string `arg:"positional,required" help:"新名称" placeholder:"新名称"`
}

func (cmd *groupSessionRenameCmd) Run(ctx *gonebot.Context, plugin *ChatgptPlugin) {
	groupId := ctx.Event.(*gonebot.GroupMessageEvent).GroupId
	if session := plugin.sessionManager.GetGroupSessionByIdOrName(groupId, cmd.NewName); session != nil {
		ctx.Replyf("会话名称已被使用，该会话ID为%s", session.SessionId)
		return
	}

	if cmd.SessionId == "" {
		session := plugin.sessionManager.GetGroupCurrentSession(groupId)
		session.SessionName = cmd.NewName
		ctx.Replyf("已重命名当前会话为%s，可使用该名称代替ID", cmd.NewName)
	} else {
		plugin.sessionManager.SetSessionName(cmd.SessionId, cmd.NewName)
		ctx.Replyf("已重命名会话%s为%s，可使用该名称代替ID", cmd.SessionId, cmd.NewName)
	}
}
