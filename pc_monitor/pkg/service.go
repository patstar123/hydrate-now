package pkg

import (
	"fmt"
	"github.com/kardianos/service"
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	"os"
)

// 以服务形式后台运行，但在消息通知方面遇到了些问题，代码暂存后面再解决

func RunService(loadBuilding func()) {
	fmt.Println("!!! Run as service")

	RedirectLogToFile()
	loadBuilding()
	base.InitDefaultLogger()

	service := createService()
	reminder := GetHNReminder()
	defer reminder.Release()

	sender := NewNotificationSender(AppName)

	configFile := getConfigFilePath()
	if true {
		if res := reminder.Init(configFile, nil, sender); !res.IsOk() {
			logger.Warnw("reminder init with error", res)
			return
		}
	} else {
		lg, err := service.Logger(nil)
		if err != nil {
			logger.Warnw("failed to get service logger", err)
			return
		}

		sl := NewServiceLogger(lg, "info")
		if res := reminder.Init(configFile, sl, sender); !res.IsOk() {
			logger.Warnw("reminder init with error", res)
			return
		}
	}

	err := service.Run()
	if err != nil && err.Error() != "(0) " {
		logger.Warnw("failed to run service", err)
		os.Exit(-1)
		return
	}

	logger.Infow("exit service")
}

func InstallService(loadBuilding func()) {
	service := createService()
	err := service.Install()
	if err != nil {
		logger.Warnw("Failed to install service", err)
	} else {
		logger.Infow("Service installed")
	}
}

func UninstallService(loadBuilding func()) {
	service := createService()
	err := service.Uninstall()
	if err != nil {
		logger.Warnw("Failed to uninstall service", err)
	} else {
		logger.Infow("Service uninstalled")
	}
}

func StartService(loadBuilding func()) {
	service := createService()
	err := service.Start()
	if err != nil {
		logger.Warnw("Failed to start service", err)
	} else {
		logger.Infow("Service started")
	}
}

func StopService(loadBuilding func()) {
	service := createService()
	err := service.Stop()
	if err != nil {
		logger.Warnw("Failed to stop service", err)
	} else {
		logger.Infow("Service stopped")
	}
}

func RestartService(loadBuilding func()) {
	service := createService()
	err := service.Restart()
	if err != nil {
		logger.Warnw("Failed to restart service", err)
	} else {
		logger.Infow("Service restarted")
	}
}

func QueryService(loadBuilding func()) {
	service := createService()
	status, err := service.Status()
	if err != nil {
		logger.Warnw("Failed to query service status", err)
	} else {
		str, ok := status2Str[status]
		if !ok {
			str = "unknown"
		}
		logger.Infow("Service status: " + str)
	}
}

var status2Str = map[service.Status]string{
	service.StatusStopped: "stopped",
	service.StatusRunning: "running",
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
	Name:        "HydrateNowService",
	DisplayName: "Hydrate Now Service",
	Description: "A service that reminds users to stand up and drink water.",
}

type program struct{}

func (p *program) Start(s service.Service) error {
	go func() {
		res := GetHNReminder().Run()
		if !res.IsOk() {
			logger.Warnw("reminder run with error", res)
		} else {
			logger.Infow("reminder stopped")
		}
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	GetHNReminder().Release()
	return nil
}
