package main

import (
	"fmt"
	"github.com/juju/fslock"
	"github.com/livekit/protocol/logger"
	"github.com/lxn/walk"
	"lx/funny/hydrate/pc_monitor/pkg"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var IsTrayBuilding = "true"

var commands = map[string][]any{
	"": {"Run this program as service", run},

	"run": {"Run this program directly instead of as service", runDirectly},

	"autostart-on":  {"Add auto start to regedit", pkg.AddAutoStart},
	"autostart-off": {"Remove auto start from regedit", pkg.RemoveAutoStart},

	"install":   {"Install this service", pkg.InstallService},
	"uninstall": {"Uninstall this service", pkg.UninstallService},
	"start":     {"Start this service", pkg.StartService},
	"stop":      {"Stop this service", pkg.StopService},
	"restart":   {"Restart this service", pkg.RestartService},
	"status":    {"Restart this service", pkg.QueryService},
}

func printAvailableCommands() {
	fmt.Println("Available commands:")
	for cmd, action := range commands {
		fmt.Printf("  %s: %s\n", cmd, action[0].(string))
	}
}

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	action, exists := commands[command]
	if !exists {
		fmt.Printf("Unknown command: %s\n", command)
		printAvailableCommands()
		os.Exit(1)
	}

	action[1].(func(loadBuilding func()))(func() {
		loadBuilding()
	})
}

func run(loadBuilding func()) {
	lock := getAppLock()
	err := lock.TryLock()
	if err != nil {
		walk.MsgBox(nil, pkg.AppName, "请勿重复运行", walk.MsgBoxOK)
		return
	}
	defer lock.Unlock()

	if IsTrayBuilding == "true" {
		pkg.RunAsTray(loadBuilding)
	} else {
		pkg.RunService(loadBuilding)
	}
}

func runDirectly(loadBuilding func()) {
	lock := getAppLock()
	err := lock.TryLock()
	if err != nil {
		walk.MsgBox(nil, pkg.AppName, "请勿重复运行", walk.MsgBoxOK)
		return
	}
	defer lock.Unlock()

	loadBuilding()

	reminder := pkg.GetHNReminder()
	defer reminder.Release()

	sender := pkg.NewMessageBoxSender(pkg.AppName)

	if res := reminder.Init(pkg.ConfigFileName, nil, sender); !res.IsOk() {
		logger.Warnw("reminder init with error", res)
		return
	}

	go func() {
		if res := reminder.Run(); !res.IsOk() {
			logger.Warnw("reminder run with error", res)
			return
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-done
}

func getAppLock() *fslock.Lock {
	lockPath := filepath.Join(os.TempDir(), "hydrate_now.lock")
	logger.Infow("check app lock file: " + lockPath)
	return fslock.New(lockPath)
}
