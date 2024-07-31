package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	bu "github.com/patstar123/go-base/utils"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ConfigFileName = "config.yaml"
	AppName        = "HydrateNow"
)

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

	logger.Infow("loadConfigFile", "config", r.config, "file", configFile)

	r.initHttp()
	r.msgSender = msgSender

	r.shouldRemind = false
	lastBreakTime := getLastBreakTimeFromTemp()
	if lastBreakTime != nil {
		r.lastBreakTime = *lastBreakTime
	} else {
		r.resetLastBreakTime()
	}

	logger.Infow("init successfully", "lastBreakTime", r.lastBreakTime)
	return base.SUCCESS
}

func (r *HNReminder) Run() base.Result {
	r.running = true
	defer func() { r.running = false }()

	go r.remindingCheckLoop()
	go r.connect2Router()

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

func (r *HNReminder) GetStatus() (shouldRemind bool, nextDuration time.Duration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	if r.shouldRemind {
		return true, r.lastRemindTime.Add(time.Duration(r.config.AlwaysRemindIntervalSec) * time.Second).Sub(now)
	} else {
		return false, r.lastBreakTime.Add(time.Duration(r.config.BreakIntervalSec) * time.Second).Sub(now)
	}
}

func (r *HNReminder) initHttp() {
	r.http = bu.CreateGinHttp(nil)
	r.http.Any("/reset_remind", r.onReqResetRemindHandler)
}

func (r *HNReminder) connect2Router() {
	if r.config.RouterUrl == "" {
		logger.Warnw("there is no route url, so it would work in standalone mode", nil)
		return
	}

	logger.Infow("Connecting to router: " + r.config.RouterUrl)
	c, _, err := websocket.DefaultDialer.Dial(r.config.RouterUrl, nil)
	if err != nil {
		logger.Warnw("ws dial failed", err)
		r.delay2ReconnectRouter()
		return
	}
	defer c.Close()

	err = c.WriteJSON(r.config.ClientId)
	if err != nil {
		logger.Warnw("ws write client id failed", err)
		r.delay2ReconnectRouter()
		return
	}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			logger.Warnw("ws read error", err)
			r.delay2ReconnectRouter()
			return
		}

		logger.Debugw("ws received: " + string(message))
		if string(message) == "reset_remind" {
			err = c.WriteMessage(websocket.TextMessage, []byte("Good boy"))
			if err != nil {
				logger.Warnw("ws write rsp failed", err)
				r.delay2ReconnectRouter()
				return
			}
			r.resetRemind()
		}
	}
}

func (r *HNReminder) onReqResetRemindHandler(c *gin.Context) {
	bu.LogHttpRequest(nil)
	bu.ReturnRsp(c, http.StatusOK, "Good boy")
	r.resetRemind()
}

func (r *HNReminder) delay2ReconnectRouter() {
	go func() {
		time.Sleep(5 * time.Second)
		r.connect2Router()
	}()
}

func (r *HNReminder) resetRemind() {
	logger.Infow("HydrateNow: good boy")

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.shouldRemind = false
	r.resetLastBreakTime()

	if r.shownRemind {
		go r.closeReminder()
	}
}

type _Config struct {
	BreakIntervalSec        int    `yaml:"break_interval_sec"`
	AlwaysRemindIntervalSec int    `yaml:"always_remind_interval_sec"`
	ApiPort                 string `yaml:"api_port"`

	ClientId  string `yaml:"client_id"`
	RouterUrl string `yaml:"router_url"`

	Logging logger.Config `yaml:"logging,omitempty" json:"-"`
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

	if r.config.ClientId == "" {
		logger.Warnw("not config client_id", nil)
		return base.INVALID_PARAM
	}

	if r.config.RouterUrl == "" {
		logger.Warnw("not config router_url", nil)
	}

	return base.SUCCESS
}

func (r *HNReminder) remindingCheckLoop() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()

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
		//logger.Warnw("HydrateNow: shouldRemind", nil)
		if now.Sub(r.lastRemindTime) > time.Duration(r.config.AlwaysRemindIntervalSec)*time.Second {
			//logger.Warnw("HydrateNow: showReminder", nil)
			r.showReminder()
		}
	} else {
		//logger.Warnw("HydrateNow: check BreakTime", nil)
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

func (r *HNReminder) resetLastBreakTime() {
	r.lastBreakTime = time.Now()
	saveLastBreakTimeToTemp(&r.lastBreakTime)
}

func getLastBreakTimeFromTemp() *time.Time {
	lastFile := filepath.Join(os.TempDir(), "hydrate_now.last_break")
	if _, err := os.Stat(lastFile); os.IsNotExist(err) {
		return nil
	}

	content, err := ioutil.ReadFile(lastFile)
	if err != nil {
		logger.Warnw("failed to read last break time from temp file", err)
		return nil
	}

	lastTime := &time.Time{}
	err = lastTime.UnmarshalText(content)
	if err != nil {
		logger.Warnw("failed to UnmarshalText for last break time", err)
		return nil
	}

	return lastTime
}

func saveLastBreakTimeToTemp(time *time.Time) {
	content, err := time.MarshalText()
	if err != nil {
		logger.Warnw("failed to MarshalText for last break time", err)
		return
	}

	lastFile := filepath.Join(os.TempDir(), "hydrate_now.last_break")
	err = ioutil.WriteFile(lastFile, content, 0644)
	if err != nil {
		logger.Warnw("failed to update last break time to temp file", err)
		return
	}
}
