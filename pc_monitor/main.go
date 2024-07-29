package main

import (
	"fmt"
	"github.com/livekit/protocol/logger"
	"lx/funny/hydrate/pc_monitor/pkg"
	"os"
	"os/signal"
	"syscall"
)

var commands = map[string][]any{
	"run": {"Run this program directly instead of as service", runDirectly},

	"":          {"Run this program as service", pkg.RunService},
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

func runDirectly(loadBuilding func()) {
	loadBuilding()

	reminder := pkg.GetHNReminder()
	defer reminder.Release()

	sender := pkg.NewMessageBoxSender("HydrateNow")

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
