package pluginmanage

import (
	"fmt"
	"strings"

	"github.com/liwh011/gonebot"
)

func init() {
	pm := &PluginManager{
		pluginStates: make(map[string]*PluginState),
	}
	gonebot.RegisterPlugin(pm, nil)

	gonebot.EngineHookManager.PluginWillLoad(func(ph *gonebot.PluginHub) {
		pluginId := ph.GetPluginId()
		pm.pluginStates[pluginId] = newPluginState()

		// 添加一个中间件来检查插件是否被禁用
		ph.Use(func(ctx *gonebot.Context) bool {
			if ev, ok := ctx.Event.(*gonebot.GroupMessageEvent); ok {
				enabled, _ := pm.IsPluginEnabled(pluginId, ev.GroupId)
				return enabled
			}
			return true
		})
	})
}

type groupSet []int64

func (set *groupSet) Add(groupId int64) {
	for _, id := range *set {
		if id == groupId {
			return
		}
	}
	*set = append(*set, groupId)
}

func (set *groupSet) Remove(groupId int64) {
	for i, id := range *set {
		if id == groupId {
			*set = append((*set)[:i], (*set)[i+1:]...)
			return
		}
	}
}

func (set *groupSet) Contains(groupId int64) bool {
	for _, id := range *set {
		if id == groupId {
			return true
		}
	}
	return false
}

type PluginState struct {
	enabled         bool
	enableOnDefault bool
	enabledGroups   groupSet
	disabledGroups  groupSet
	visible         bool
}

func newPluginState() *PluginState {
	return &PluginState{
		enabled:         true,
		enableOnDefault: true,
		enabledGroups:   groupSet{},
		disabledGroups:  groupSet{},
		visible:         true,
	}
}

func (state *PluginState) IsEnabled(groupId int64) bool {
	if !state.enabled {
		return false
	}

	for _, g := range state.disabledGroups {
		if g == groupId {
			return false
		}
	}
	for _, g := range state.enabledGroups {
		if g == groupId {
			return true
		}
	}
	return state.enableOnDefault
}

type PluginManager struct {
	pluginStates map[string]*PluginState
}

func (pm *PluginManager) GetPluginInfo() gonebot.PluginInfo {
	return gonebot.PluginInfo{
		Name:        "plugin_manage",
		Description: "提供插件的动态、按群组管理",
		Author:      "liwh011",
		Version:     "0.0.1",
	}
}

func (pm *PluginManager) Init(hub *gonebot.PluginHub) {
	// 群内，管理员可用
	hub.NewHandler(gonebot.EventNameGroupMessage).
		Use(gonebot.FromAdminOrHigher()).
		Use(gonebot.ShellLikeCommand("plugin", groupCommands{}, gonebot.ParseFailedAction_AutoReply)).
		Handle(func(ctx *gonebot.Context) {
			parseResult := ctx.GetShellLikeCommandResult()
			if !parseResult.HasSubcommand() {
				msg := parseResult.FormatErrorAndHelp(fmt.Errorf("未指定子命令"))
				ctx.Reply(msg)
				return
			}
			parseResult.GetSubcommand().(Command).Run(ctx, pm)
		})

	// 私聊，仅超管可用
	hub.NewHandler(gonebot.EventNamePrivateMessage).
		Use(gonebot.FromSuperuser()).
		Use(gonebot.ShellLikeCommand("plugin", privateCommands{}, gonebot.ParseFailedAction_AutoReply)).
		Handle(func(ctx *gonebot.Context) {
			parseResult := ctx.GetShellLikeCommandResult()
			if !parseResult.HasSubcommand() {
				msg := parseResult.FormatErrorAndHelp(fmt.Errorf("未指定子命令"))
				ctx.Reply(msg)
				return
			}
			parseResult.GetSubcommand().(Command).Run(ctx, pm)
		})
}

// 某插件是否在某群启用
func (pm *PluginManager) IsPluginEnabled(pluginId string, groupId int64) (bool, error) {
	state, ok := pm.pluginStates[pluginId]
	if !ok {
		return false, fmt.Errorf("插件%s不存在", pluginId)
	}
	return state.IsEnabled(groupId), nil
}

// 分别列出在某群启用和禁用的插件
func (pm *PluginManager) ListPlugins(groupId int64) (enabled, disabled []string) {
	for id, state := range pm.pluginStates {
		if state.visible {
			if state.IsEnabled(groupId) {
				enabled = append(enabled, id)
			} else {
				disabled = append(disabled, id)
			}
		}
	}
	return
}

// 分别列出启用和禁用某插件的群
func (pm *PluginManager) GetEnabledGroupsOfPlugin(pluginId string, groupIds []int64) (enabled, disabled []int64) {
	state, ok := pm.pluginStates[pluginId]
	if !ok {
		return
	}
	for _, id := range groupIds {
		if state.IsEnabled(id) {
			enabled = append(enabled, id)
		} else {
			disabled = append(disabled, id)
		}
	}
	return
}

// 在某群启用/禁用某些插件。返回成功的数量和错误信息（如找不到某些插件）
func (pm *PluginManager) EnablePlugin(pluginIds []string, groupId int64, enable bool) (cnt int, err error) {
	invalidPluginIds := []string{}
	for _, id := range pluginIds {
		state, ok := pm.pluginStates[id]
		if !ok {
			invalidPluginIds = append(invalidPluginIds, id)
			continue
		}

		if enable {
			state.enabledGroups.Add(groupId)
			state.disabledGroups.Remove(groupId)
		} else {
			state.enabledGroups.Remove(groupId)
			state.disabledGroups.Add(groupId)
		}
		cnt++
	}

	if len(invalidPluginIds) > 0 {
		err = fmt.Errorf("没有找到以下插件ID：%s", strings.Join(invalidPluginIds, ", "))
	}
	return
}
