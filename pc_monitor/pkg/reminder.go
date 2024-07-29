package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	bu "github.com/patstar123/go-base/utils"
	"net/http"
	"sync"
	"time"
)

const ConfigFileName = "config.yaml"

type HNReminder struct {
	config    _Config
	http      *gin.Engine
	running   bool
	msgSender MessageSender

	mutex          sync.Mutex
	lastBreakTime  time.Time
	lastRemindTime time.Time
	shouldRemind   bool
	shownRemind    bool
}

var (
	gReminder = &HNReminder{
		running:      false,
		mutex:        sync.Mutex{},
		shouldRemind: false,
		shownRemind:  false,
	}
)

func GetHNReminder() *HNReminder {
	return gReminder
}

func (r *HNReminder) Init(configFile string, external *ServiceLogger, msgSender MessageSender) base.Result {
	res := r.loadConfigFile(configFile)
	if !res.IsOk() {
		return res
	}

	// 从配置初始化全局PkgLogger
	base.InitLogger("hn", &r.config.Logging)
	if external != nil {
		_, c, err := logger.NewZapLogger(&r.config.Logging)
		if err == nil {
			external.SetLevel(r.config.Logging.Level)
			logger.SetLogger(external, c, "hn")
		}
	}

	logger.Warnw("loadConfigFile", nil, "config", r.config, "file", configFile)

	r.initHttp()
	r.msgSender = msgSender

	logger.Infow("init successfully")
	return base.SUCCESS
}

func (r *HNReminder) Run() base.Result {
	r.running = true
	defer func() { r.running = false }()
	go r.remindingCheckLoop()

	logger.Infow("run in http loop")
	err := r.http.Run(":" + r.config.ApiPort)
	if err != nil {
		return base.INTERNAL_ERROR.AppendErr("run http server error", err)
	}
	logger.Infow("exit from http loop")

	return base.SUCCESS
}

func (r *HNReminder) Release() base.Result {
	r.running = false
	if r.http != nil {
		logger.Infow("try to release")
		r.http = nil
		logger.Infow("released")
	}
	return base.SUCCESS
}

func (r *HNReminder) initHttp() {
	r.http = bu.CreateGinHttp(nil)
	r.http.Any("/reset_remind", r.onReqResetRemindHandler)
}

func (r *HNReminder) onReqResetRemindHandler(c *gin.Context) {
	bu.LogHttpRequest(nil)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.shouldRemind = false
	r.lastBreakTime = time.Now()
	bu.ReturnRsp(c, http.StatusOK, "Good boy")

	logger.Infow("HydrateNow: good boy")
	if r.shownRemind {
		go r.closeReminder()
	}
}

type _Config struct {
	BreakIntervalSec        int    `yaml:"break_interval_sec"`
	AlwaysRemindIntervalSec int    `yaml:"always_remind_interval_sec"`
	ApiPort                 string `yaml:"api_port"`

	Logging logger.Config `yaml:"logging,omitempty"`
}

func (r *HNReminder) loadConfigFile(configFile string) base.Result {
	res := bu.GetConfig(configFile, &r.config)
	if !res.IsOk() {
		return res
	}

	if r.config.BreakIntervalSec <= 0 {
		r.config.BreakIntervalSec = 1 * 60 * 60
	}

	if r.config.AlwaysRemindIntervalSec <= 0 {
		r.config.AlwaysRemindIntervalSec = 5
	}

	if r.config.ApiPort == "" {
		r.config.ApiPort = "18081"
	}

	return base.SUCCESS
}

func (r *HNReminder) remindingCheckLoop() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	r.shouldRemind = false
	r.lastBreakTime = time.Now()
	for r.running {
		select {
		case <-timer.C:
			r.checkReminder()
		}
	}
}

func (r *HNReminder) checkReminder() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	if r.shouldRemind {
		logger.Warnw("HydrateNow: shouldRemind", nil)
		if now.Sub(r.lastRemindTime) > time.Duration(r.config.AlwaysRemindIntervalSec)*time.Second {
			logger.Warnw("HydrateNow: showReminder", nil)
			r.showReminder()
		}
	} else {
		logger.Warnw("HydrateNow: check BreakTime", nil)
		if now.Sub(r.lastBreakTime) > time.Duration(r.config.BreakIntervalSec)*time.Second {
			logger.Infow("HydrateNow: break time")
			r.shouldRemind = true
			r.showReminder()
		}
	}
}

const message = "你已经工作了一段时间，请站起来去喝水。"

func (r *HNReminder) showReminder() {
	if r.shownRemind {
		return
	}

	r.shownRemind = true
	go func() {
		r.msgSender.Show(message)

		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.lastRemindTime = time.Now()
		r.shownRemind = false
	}()
}

func (r *HNReminder) closeReminder() {
	if r.shownRemind {
		r.msgSender.Close()
	}
}
