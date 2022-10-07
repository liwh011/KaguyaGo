package nonebotinteract

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	// "os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/liwh011/gonebot"
	"github.com/sirupsen/logrus"
)

type NonebotHost struct {
	pythonPath   string
	wsServerPort int
	process      *os.Process
	processState *os.ProcessState
}

func NewNonebotHost(pythonPath string) *NonebotHost {
	return &NonebotHost{
		pythonPath:   pythonPath,
		wsServerPort: 11451,
	}
}

func (host *NonebotHost) CheckPythonVersion() error {
	cmd := exec.Command(host.pythonPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	match := regexp.MustCompile(`^Python 3\.(\d+)\.\d+$`).FindStringSubmatch(strings.TrimSpace(string(out)))
	if match == nil {
		return fmt.Errorf("Python版本不符合要求，当前版本为%s", out)
	}

	ver, _ := strconv.Atoi(match[1])
	if ver < 8 {
		return fmt.Errorf("Python版本不符合要求，当前版本为%s", out)
	}

	return nil
}

func (host *NonebotHost) CheckPythonPackage() error {
	// TODO: 检查Python包是否安装
	return nil
}

func (host *NonebotHost) Start() error {
	if host.process != nil {
		if !host.processState.Exited() {
			return fmt.Errorf("nonebot已经在运行")
		}
	}

	script := fmt.Sprintf(`
# coding=UTF-8

import nonebot
from nonebot.log import logger
from nonebot.adapters.onebot.v11 import Adapter

nonebot.init(
	driver="~httpx+~websockets",
	onebot_ws_urls=["ws://127.0.0.1:%d/"],
)
driver = nonebot.get_driver()
driver.register_adapter(Adapter)

@driver.on_bot_disconnect
async def disconnect():
    logger.info("Bot disconnected. Terminating...")
    exit(0)

nonebot.run()
	`, host.wsServerPort)

	cmd := exec.Command(host.pythonPath, "-c", script)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	err = cmd.Start()
	if err != nil {
		return err
	}
	host.process = cmd.Process
	host.processState = cmd.ProcessState

	gonebot.EngineHookManager.EngineWillTerminate(func(e *gonebot.Engine) {
		logrus.Info("正在停止nonebot...")
		err := host.Stop()
		if err != nil {
			logrus.Errorf("停止nonebot失败: %s", err)
			return
		}
		logrus.Info("nonebot已停止")
	})

	return nil
}

func (host *NonebotHost) Stop() error {
	if host.process == nil {
		return fmt.Errorf("nonebot未在运行")
	}

	err := host.process.Signal(syscall.SIGINT)
	if err != nil {
		err = host.process.Kill()
	}

	return err
}
