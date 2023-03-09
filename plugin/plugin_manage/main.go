package pluginmanage

import (
	"fmt"
	"strings"

	"github.com/liwh011/gonebot"
	"github.com/sirupsen/logrus"
)

func init() {
	pm := &PluginManager{
		pluginStates: make(map[string]*PluginState),
	}
	gonebot.RegisterPlugin(pm, nil)

	gonebot.GlobalHooks.PluginWillLoad(func(ph *gonebot.PluginHub) {
		pluginId := ph.GetPluginId()
		pm.pluginStates[pluginId] = newPluginState()

		// 添加一个中间件来检查插件是否被禁用
		ph.Use(func(ctx *gonebot.Context) bool {
			if pluginId == "plugin_manage@liwh011" {
				return true
			}
			if _, ok := ctx.Event.(*gonebot.PrivateMessageEvent); ok {
				return pm.IsPluginEnabledGlobally(pluginId)
			} else if ev, ok := ctx.Event.(*gonebot.GroupMessageEvent); ok {
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
	Enabled         bool
	EnableOnDefault bool
	EnabledGroups   groupSet
	EisabledGroups  groupSet
	Visible         bool
}

func newPluginState() *PluginState {
	return &PluginState{
		Enabled:         true,
		EnableOnDefault: true,
		EnabledGroups:   groupSet{},
		EisabledGroups:  groupSet{},
		Visible:         true,
	}
}

func (state *PluginState) IsEnabled(groupId int64) bool {
	if !state.Enabled {
		return false
	}

	for _, g := range state.EisabledGroups {
		if g == groupId {
			return false
		}
	}
	for _, g := range state.EnabledGroups {
		if g == groupId {
			return true
		}
	}
	return state.EnableOnDefault
}

type PluginManager struct {
	pluginStates map[string]*PluginState
	storage      gonebot.Storage
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
	var err error
	pm.storage, err = gonebot.NewStorage("plugin_manage")
	if err != nil {
		logrus.Fatalf("无法创建或获取plugin_manage插件的存储: %v", err)
	}
	pm.restoreStates()

	// 群内，管理员可用
	hub.NewHandler(gonebot.EventName_GroupMessage).
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
	hub.NewHandler(gonebot.EventName_PrivateMessage).
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

func (pm *PluginManager) IsPluginEnabledGlobally(pluginId string) bool {
	state, ok := pm.pluginStates[pluginId]
	if !ok {
		return false
	}
	return state.Enabled
}

// 某插件是否在某群启用
func (pm *PluginManager) IsPluginEnabled(pluginId string, groupId int64) (bool, error) {
	state, ok := pm.pluginStates[pluginId]
	if !ok {
		return false, fmt.Errorf("插件%s不存在", pluginId)
	}
	return state.IsEnabled(groupId), nil
}

func (pm *PluginManager) storeStates() {
	err := pm.storage.Set("plugin_states", "plugin_states", pm.pluginStates)
	if err != nil {
		logrus.Warnf("无法保存插件状态，重启进程将导致未保存状态丢失: %v", err)
	}
}

func (pm *PluginManager) restoreStates() {
	err := pm.storage.Get("plugin_states", "plugin_states", &pm.pluginStates)
	// 如果存储中没有数据，就使用默认值
	if err != nil {
		pm.storage.Set("plugin_states", "plugin_states", pm.pluginStates)
	}
}

// 分别列出在某群启用和禁用的插件
func (pm *PluginManager) ListPlugins(groupId int64) (enabled, disabled []string) {
	for id, state := range pm.pluginStates {
		if state.Visible {
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
			state.EnabledGroups.Add(groupId)
			state.EisabledGroups.Remove(groupId)
		} else {
			state.EnabledGroups.Remove(groupId)
			state.EisabledGroups.Add(groupId)
		}
		cnt++
	}
	pm.storeStates()

	if len(invalidPluginIds) > 0 {
		err = fmt.Errorf("没有找到以下插件ID：%s", strings.Join(invalidPluginIds, ", "))
	}
	return
}
