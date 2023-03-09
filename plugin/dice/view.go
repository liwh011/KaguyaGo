package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"

	"github.com/liwh011/gonebot"
)

func init() {
	gonebot.RegisterPlugin(&DicePlugin{}, nil)
}

type DicePlugin struct{}

func (DicePlugin) GetPluginInfo() gonebot.PluginInfo {
	return gonebot.PluginInfo{
		Name:        "dice",
		Description: "跑团骰子",
		Version:     "0.0.1",
		Author:      "liwh011",
	}
}

func (DicePlugin) Init(engine *gonebot.PluginHub) {
	checkMw := func(ctx *gonebot.Context) bool {
		params, err := parseText(ctx.Event.ExtractPlainText())
		if err != nil {
			ctx.Reply(err.Error())
			ctx.Abort()
			return false
		}
		ctx.Set("dice", params)
		return true
	}

	engine.
		NewHandler(gonebot.EventName_GroupMessage).
		Use(gonebot.StartsWith(".r")).
		Use(checkMw).
		Handle(func(ctx *gonebot.Context) {
			params := ctx.MustGet("dice").(diceParams)
			res := doDice(params)
			ctx.Reply("你的投掷结果是", res)
		})
}

type diceParams struct {
	num    int // 数量
	min    int // 最小值
	max    int // 最大值
	sign   int // 偏移的符号
	offset int // 偏移
}

func parseText(text string) (params diceParams, err error) {
	dicePat := regexp.MustCompile(`^\.r\s*((?P<num>\d*)d((?P<min>\d+)~)?(?P<max>\d*)((?P<sign>[+-])(?P<offset>\d*))?)?$`)
	matchGroup := dicePat.FindStringSubmatch(text)
	if matchGroup == nil {
		err = fmt.Errorf("投骰子的格式为：.r <数量>d[最小值~]<最大值>[+|-偏移量]，例如：.r 3d6+1")
		return
	}

	matchMap := make(map[string]string)
	for i, name := range dicePat.SubexpNames() {
		if i == 0 {
			continue
		}
		matchMap[name] = matchGroup[i]
	}

	// 默认值
	params = diceParams{
		num:    1,
		min:    1,
		max:    100,
		sign:   1,
		offset: 0,
	}
	if matchMap["num"] != "" {
		if len(matchMap["num"]) > 2 {
			err = fmt.Errorf("骰子数量不能超过99")
			return
		}
		params.num, _ = strconv.Atoi(matchMap["num"])
	}
	if matchMap["min"] != "" {
		if len(matchMap["min"]) > 4 {
			err = fmt.Errorf("最小值不能超过9999")
			return
		}
		params.min, _ = strconv.Atoi(matchMap["min"])
	}
	if matchMap["max"] != "" {
		if len(matchMap["max"]) > 4 {
			err = fmt.Errorf("最大值不能超过9999")
			return
		}
		params.max, _ = strconv.Atoi(matchMap["max"])
	}
	if matchMap["sign"] != "" {
		if matchMap["sign"] == "-" {
			params.sign = -1
		} else {
			params.sign = 1
		}
	}
	if matchMap["offset"] != "" {
		if len(matchMap["offset"]) > 5 {
			err = fmt.Errorf("偏移量不能超过99999")
			return
		}
		params.offset, _ = strconv.Atoi(matchMap["offset"])
	}

	return
}

type diceResult struct {
	params diceParams
	rolls  []int // 所有色子的点数
	sum    int   // 所有色子的点数之和
	res    int   // 结果
}

func (res diceResult) String() string {
	if res.params.num == 0 {
		return ""
	}

	num := fmt.Sprintf("%dD", res.params.num)
	minmax := fmt.Sprintf("%d~%d", res.params.min, res.params.max)
	if res.params.min == 1 {
		minmax = fmt.Sprintf("%d", res.params.max)
	}
	offset := fmt.Sprintf("%+d", res.params.offset*res.params.sign)
	if res.params.offset == 0 {
		offset = ""
	}
	rolls := ""
	for i, r := range res.rolls {
		if i != 0 {
			rolls += "+"
		}
		rolls += fmt.Sprintf("%d", r)
	}

	return fmt.Sprintf("%s%s%s = %s%s = %d", num, minmax, offset, rolls, offset, res.res)
}

func doDice(params diceParams) diceResult {
	if params.min > params.max {
		params.min, params.max = params.max, params.min
	}

	if params.num == 0 {
		return diceResult{
			params: params,
		}
	}

	rolls := make([]int, params.num)
	sum := 0
	for i := 0; i < params.num; i++ {
		rolls[i] = rand.Intn(params.max-params.min+1) + params.min
		sum += rolls[i]
	}

	res := sum + params.sign*params.offset

	return diceResult{
		params: params,
		rolls:  rolls,
		sum:    sum,
		res:    res,
	}
}
