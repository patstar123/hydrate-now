package main

import (
	"fmt"
	"github.com/kardianos/service"
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	"lx/funny/hydrate/pc_monitor/pkg"
	"os"
	"os/signal"
	"syscall"
)

var commands = map[string][]any{
	"":          {"Run this program as service", runService},
	"run":       {"Run this program directly instead of as service", runDirectly},
	"install":   {"Install this service", installService},
	"uninstall": {"Uninstall this service", uninstallService},
	"start":     {"Start this service", startService},
	"stop":      {"Stop this service", stopService},
	"restart":   {"Restart this service", restartService},
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

	loadBuilding()
	base.InitDefaultLogger()

	action[1].(func())()
}

func runDirectly() {
	reminder := pkg.GetHNReminder()
	defer reminder.Release()

	if res := reminder.Init(nil); !res.IsOk() {
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

func runService() {
	service := createService()
	lg, err := service.Logger(nil)
	if err != nil {
		logger.Warnw("failed to get service logger", err)
		return
	}

	reminder := pkg.GetHNReminder()
	defer reminder.Release()

	sl := pkg.NewServiceLogger(lg, "info")
	if res := reminder.Init(sl); !res.IsOk() {
		logger.Warnw("reminder init with error", res)
		return
	}

	err = service.Run()
	if err != nil {
		logger.Warnw("failed to run service", err)
		return
	}
}

func installService() {
	service := createService()
	err := service.Install()
	if err != nil {
		logger.Warnw("Failed to install service", err)
	} else {
		logger.Infow("Service installed")
	}
}

func uninstallService() {
	service := createService()
	err := service.Uninstall()
	if err != nil {
		logger.Warnw("Failed to uninstall service", err)
	} else {
		logger.Infow("Service uninstalled")
	}
}

func startService() {
	service := createService()
	err := service.Start()
	if err != nil {
		logger.Warnw("Failed to start service", err)
	} else {
		logger.Infow("Service started")
	}
}

func stopService() {
	service := createService()
	err := service.Stop()
	if err != nil {
		logger.Warnw("Failed to stop service", err)
	} else {
		logger.Infow("Service stopped")
	}
}

func restartService() {
	service := createService()
	err := service.Restart()
	if err != nil {
		logger.Warnw("Failed to restart service", err)
	} else {
		logger.Infow("Service restarted")
	}
}

func createService() service.Service {
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Warnw("create service failed", err)
		os.Exit(1)
	}
	return s
}

var svcConfig = &service.Config{
	Name:        "HydrateReminderService",
	DisplayName: "Hydrate Reminder Service",
	Description: "A service that reminds users to stand up and drink water.",
}

type program struct{}

func (p *program) Start(s service.Service) error {
	go func() {
		res := pkg.GetHNReminder().Run()
		if !res.IsOk() {
			logger.Warnw("reminder run with error", res)
		} else {
			logger.Infow("reminder stopped")
		}
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	return pkg.GetHNReminder().Release()
}
