package pkg

import (
	"fmt"
	"github.com/getlantern/systray"
	"github.com/livekit/protocol/logger"
	"github.com/lxn/walk"
	"github.com/patstar123/go-base"
	"golang.org/x/sys/windows/registry"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func RunAsTray(loadBuilding func()) {
	fmt.Println("!!! Run as tray")

	RedirectLogToFile()
	loadBuilding()
	base.InitDefaultLogger()

	reminder := GetHNReminder()
	defer reminder.Release()

	sender := NewMessageBoxSender(AppName)

	configFile := getConfigFilePath()
	if res := reminder.Init(configFile, nil, sender); !res.IsOk() {
		logger.Warnw("reminder init with error", res)
		return
	}

	err := setAutoStart(AppName, true)
	if err != nil {
		logger.Warnw("setAutoStart failed", err)
	}

	systray.Run(onReady, onExit)
}

func AddAutoStart(loadBuilding func()) {
	base.InitDefaultLogger()

	err := setAutoStart(AppName, false)
	if err != nil {
		logger.Warnw("AddAutoStart failed", err)
	} else {
		logger.Infow("success to AddAutoStart")
	}
}

func RemoveAutoStart(loadBuilding func()) {
	base.InitDefaultLogger()

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		logger.Warnw("registry.OpenKey failed", err)
		return
	}
	defer key.Close()

	err = key.DeleteValue(AppName)
	if err != nil {
		logger.Warnw("registry.DeleteValue failed", err)
		return
	}

	logger.Infow("success to RemoveAutoStart")
	return
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle(AppName)
	systray.SetTooltip("多走动多喝水")

	// 添加菜单项和处理方法
	mQuit := systray.AddMenuItem("退出", "退出程序")
	go func() {
		for {
			<-mQuit.ClickedCh
			if true {
				walk.MsgBox(nil, AppName, "不允许退出(除非你Kill它)", walk.MsgBoxOK)
				continue
			}
			break
		}

		logger.Infow("quit by user")
		systray.Quit()
	}()

	go func() {
		if res := GetHNReminder().Run(); !res.IsOk() {
			logger.Warnw("reminder run with error", res)
			return
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			shouldRemind, nextDuration := GetHNReminder().GetStatus()
			seconds := int(nextDuration.Seconds()) % 3600 % 60
			minutes := int(nextDuration.Minutes()) % 60
			hours := int(nextDuration.Hours())
			hit := ""
			if hours > 0 {
				hit += strconv.Itoa(hours) + "小时"
			}
			if minutes > 0 {
				hit += strconv.Itoa(minutes) + "分钟"
			}
			if hours <= 0 && seconds > 0 {
				hit += strconv.Itoa(seconds) + "秒"
			}

			if shouldRemind {
				systray.SetTooltip(fmt.Sprintf("多走动多喝水(请尽快打卡,距离下次提醒还有%v)", hit))
			} else {
				systray.SetTooltip(fmt.Sprintf("多走动多喝水(距离下次休息还有%v)", hit))
			}
		}
	}()
}

func onExit() {
	logger.Infow("tray exited")
	GetHNReminder().Release()
}

func getIcon() []byte {
	data, err := ioutil.ReadFile(getIconFilePath())
	if err != nil {
		logger.Warnw("failed to get favicon.ico", err)
		return []byte{}
	}
	return data
}

func setAutoStart(appName string, ask bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	existingPath, _, err := key.GetStringValue(appName)
	if err == nil && existingPath == exePath {
		return nil
	}

	if ask {
		result := walk.MsgBox(nil, AppName, "是否添加为开机自启？", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
		if result != walk.DlgCmdYes {
			logger.Warnw("user not allow to auto start", nil)
			return nil
		}
	}

	err = key.SetStringValue(appName, exePath)
	if err != nil {
		return err
	}

	return nil
}
