package pluginmanage

import (
	"fmt"
	"strings"

	"github.com/liwh011/gonebot"
)

type Command interface {
	Run(ctx *gonebot.Context, pm *PluginManager)
}

type groupListCmd struct{}

func (cmd *groupListCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	ev := ctx.Event.(*gonebot.GroupMessageEvent)
	enabled, disabled := pm.ListPlugins(ev.GroupId)

	rep := ""
	if len(enabled) > 0 {
		rep += fmt.Sprintf("\n✅已启用的插件：\n%s", strings.Join(enabled, "\n"))
	}
	if len(disabled) > 0 {
		if rep != "" {
			rep += "\n"
		}
		rep += fmt.Sprintf("\n❌已禁用的插件：\n%s", strings.Join(disabled, "\n"))
	}
	if rep == "" {
		rep = "没有可用的插件"
	}
	ctx.Reply(rep)
}

type groupEnableCmd struct {
	Ids []string `arg:"positional,required" help:"插件ID，形如xx@xx" placeholder:"插件ID"`
}

func (cmd *groupEnableCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	ev := ctx.Event.(*gonebot.GroupMessageEvent)
	cnt, err := pm.EnablePlugin(cmd.Ids, ev.GroupId, true)
	if err != nil {
		if cnt > 0 {
			ctx.Reply(fmt.Sprintf("成功启用%d个插件，%s", cnt, err.Error()))
		} else {
			ctx.Reply(err.Error())
		}
	} else {
		ctx.Reply(fmt.Sprintf("成功启用%d个插件", cnt))
	}
}

type groupDisableCmd struct {
	Ids []string `arg:"positional,required" help:"插件ID，形如xx@xx" placeholder:"插件ID"`
}

func (cmd *groupDisableCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	ev := ctx.Event.(*gonebot.GroupMessageEvent)
	cnt, err := pm.EnablePlugin(cmd.Ids, ev.GroupId, false)
	if err != nil {
		if cnt > 0 {
			ctx.Reply(fmt.Sprintf("成功禁用%d个插件，%s", cnt, err.Error()))
		} else {
			ctx.Reply(err.Error())
		}
	} else {
		ctx.Reply(fmt.Sprintf("成功禁用%d个插件", cnt))
	}
}

// 群组插件管理
type groupCommands struct {
	ListCmd    *groupListCmd    `arg:"subcommand:ls" help:"列出插件"`
	EnableCmd  *groupEnableCmd  `arg:"subcommand:enable" help:"启用插件"`
	DisableCmd *groupDisableCmd `arg:"subcommand:disable" help:"禁用插件"`
}

type privateListCmd struct {
	Group  int64  `arg:"-g" help:"要查看插件的群号" placeholder:"群号"`
	Plugin string `arg:"-p" help:"要查看各群启用情况的插件" placeholder:"插件ID"`
}

func (cmd *privateListCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	if cmd.Plugin == "" && cmd.Group == 0 {
		ctx.Reply("请指定要查看的插件或群号")
		return
	}

	groups, err := ctx.Bot.GetGroupList()
	if err != nil {
		ctx.Replyf("获取群列表失败：%s", err.Error())
		return
	}
	groupIdToInfo := make(map[int64]gonebot.GroupInfo)
	for _, group := range *groups {
		groupIdToInfo[group.GroupId] = group
	}

	if cmd.Group != 0 {
		targetGroupInfo, ok := groupIdToInfo[cmd.Group]
		if !ok {
			ctx.Replyf("未找到群号为%d的群", cmd.Group)
			return
		}

		if cmd.Plugin != "" {
			// 同时指定了插件和群号，查看插件在该群的启用情况
			enabled, err := pm.IsPluginEnabled(cmd.Plugin, cmd.Group)
			if err != nil {
				ctx.Replyf("查询插件%s失败：%s", cmd.Plugin, err.Error())
				return
			}
			word := "已启用"
			if !enabled {
				word = "已禁用"
			}
			rep := fmt.Sprintf("插件%s在群聊「%s (%d)」中%s", cmd.Plugin, targetGroupInfo.GroupName, targetGroupInfo.GroupId, word)
			ctx.Reply(rep)
		} else {
			// 只指定了群号，查看该群的所有插件启用情况
			enabled, disabled := pm.ListPlugins(cmd.Group)
			if len(enabled) == 0 && len(disabled) == 0 {
				ctx.Reply("没有可用的插件")
				return
			}

			rep := fmt.Sprintf("群「%s (%d)」的插件列表", targetGroupInfo.GroupName, targetGroupInfo.GroupId)
			if len(enabled) > 0 {
				rep += fmt.Sprintf("\n✅已启用的插件：\n%s\n", strings.Join(enabled, "\n"))
			}
			if len(disabled) > 0 {
				rep += fmt.Sprintf("\n❌已禁用的插件：\n%s", strings.Join(disabled, "\n"))
			}
			ctx.Reply(rep)
		}
	} else {
		// 只指定了插件，查看该插件在各群的启用情况
		groupIds := make([]int64, 0, len(*groups))
		for _, group := range *groups {
			groupIds = append(groupIds, group.GroupId)
		}
		enabled, disabled := pm.GetEnabledGroupsOfPlugin(cmd.Plugin, groupIds)
		if len(enabled) == 0 && len(disabled) == 0 {
			ctx.Reply("没有可用的插件")
			return
		}

		rep := strings.Builder{}
		rep.WriteString(fmt.Sprintf("插件%s在各群的启用情况：\n", cmd.Plugin))
		for _, groupId := range enabled {
			groupInfo, ok := groupIdToInfo[groupId]
			if !ok {
				continue
			}
			fmt.Fprintf(&rep, "✅%s (%d)\n", groupInfo.GroupName, groupInfo.GroupId)
		}
		for _, groupId := range disabled {
			groupInfo, ok := groupIdToInfo[groupId]
			if !ok {
				continue
			}
			fmt.Fprintf(&rep, "❌%s (%d)\n", groupInfo.GroupName, groupInfo.GroupId)
		}
		ctx.Reply(rep.String())
	}
}

type privateEnableCmd struct {
	Group int64    `arg:"-g,required" help:"要启用插件的群号" placeholder:"群号"`
	Ids   []string `arg:"positional,required" help:"插件ID，形如xx@xx" placeholder:"插件ID"`
}

func (cmd *privateEnableCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	cnt, err := pm.EnablePlugin(cmd.Ids, cmd.Group, true)

	if err != nil && cnt == 0 {
		ctx.Reply(err.Error())
		return
	}

	rep := &strings.Builder{}
	fmt.Fprintf(rep, "成功为群聊%d启用%d个插件。", cmd.Group, cnt)
	if err != nil {
		fmt.Fprintf(rep, "同时，%s", err.Error())
	}
	ctx.Reply(rep.String())
}

type privateDisableCmd struct {
	Group int64    `arg:"-g,required" help:"要禁用插件的群号"`
	Ids   []string `arg:"positional,required" help:"插件ID，形如xx@xx" placeholder:"插件ID"`
}

func (cmd *privateDisableCmd) Run(ctx *gonebot.Context, pm *PluginManager) {
	cnt, err := pm.EnablePlugin(cmd.Ids, cmd.Group, false)

	if err != nil && cnt == 0 {
		ctx.Reply(err.Error())
		return
	}

	rep := &strings.Builder{}
	fmt.Fprintf(rep, "成功为群聊%d禁用%d个插件。", cmd.Group, cnt)
	if err != nil {
		fmt.Fprintf(rep, "同时，%s", err.Error())
	}
	ctx.Reply(rep.String())
}

// 私聊插件管理
type privateCommands struct {
	ListCmd    *privateListCmd    `arg:"subcommand:ls" help:"列出插件"`
	EnableCmd  *privateEnableCmd  `arg:"subcommand:enable" help:"启用插件"`
	DisableCmd *privateDisableCmd `arg:"subcommand:disable" help:"禁用插件"`
}
