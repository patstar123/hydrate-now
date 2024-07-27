package pkg

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	bu "github.com/patstar123/go-base/utils"
	"net/http"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type HNReminder struct {
	config     _Config
	configFile *string
	http       *gin.Engine
	running    bool

	mutex          sync.Mutex
	lastBreakTime  time.Time
	lastRemindTime time.Time
	shouldRemind   bool
}

var (
	gReminder = &HNReminder{
		configFile:   flag.String("config", "./config.yaml", "配置文件地址"),
		running:      false,
		mutex:        sync.Mutex{},
		shouldRemind: false,
	}
)

func GetHNReminder() *HNReminder {
	return gReminder
}

func (r *HNReminder) Init(external *ServiceLogger) base.Result {
	flag.Parse()

	if r.configFile == nil {
		return base.INVALID_PARAM.AppendMsg("lost command line params: `config`")
	}

	res := r.loadConfigFile(*r.configFile)
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

	r.initHttp()

	logger.Infow("init successfully")
	return base.SUCCESS
}

func (r *HNReminder) Run() base.Result {
	logger.Infow("try to run in http loop")

	r.running = true
	go r.remindingCheckLoop()

	err := r.http.Run(":" + r.config.ApiPort)
	if err != nil {
		return base.INTERNAL_ERROR.AppendErr("run http server error", err)
	}

	r.running = false
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
	r.http.POST("/reset_remind", r.onReqResetRemindHandler)
}

func (r *HNReminder) onReqResetRemindHandler(c *gin.Context) {
	bu.LogHttpRequest(nil)
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.shouldRemind = false
	r.lastBreakTime = time.Now()
	bu.ReturnRsp(c, http.StatusOK, "Break reminder reset")
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

	logger.Infow("loadConfigFile", "config", r.config)
	return base.SUCCESS
}

func (r *HNReminder) remindingCheckLoop() {
	timer := time.NewTimer(time.Second)
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
		if now.Sub(r.lastRemindTime) > time.Duration(r.config.AlwaysRemindIntervalSec)*time.Second {
			r.lastRemindTime = now
			r.showReminder()
		}
	} else {
		if now.Sub(r.lastBreakTime) > time.Duration(r.config.BreakIntervalSec)*time.Second {
			r.shouldRemind = true
			r.lastRemindTime = now
			r.showReminder()
		}
	}
}

const (
	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_SYSTEMMODAL     = 0x00001000
)

var user32 = syscall.NewLazyDLL("user32.dll")
var msgBox = user32.NewProc("MessageBoxW")

func (r *HNReminder) showReminder() {
	title, _ := syscall.UTF16PtrFromString("休息提醒")
	message, _ := syscall.UTF16PtrFromString("你已经工作了一段时间，请站起来去喝水。")
	msgBox.Call(0,
		uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)),
		MB_OK|MB_ICONINFORMATION|MB_SYSTEMMODAL)
}
