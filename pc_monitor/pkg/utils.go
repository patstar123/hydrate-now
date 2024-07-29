package pkg

import (
	"github.com/go-toast/toast"
	"github.com/livekit/protocol/logger"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func RedirectLogToFile() {
	exePath, err := os.Executable()
	if err != nil {
		logger.Warnw("failed to get executable path", err)
		return
	}

	filename := "service-" + time.Now().Format("20060102150405") + ".log"
	logFilePath := filepath.Join(filepath.Dir(exePath), "logs", filename)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Warnw("failed to open log file", err)
		return
	}

	os.Stdout = logFile
	os.Stderr = logFile
	logger.Infow("Logging to " + logFilePath)
}

func getConfigFilePath() string {
	exePath, err := os.Executable()
	if err != nil {
		logger.Warnw("failed to get executable path", err)
		return ConfigFileName
	} else {
		return filepath.Join(filepath.Dir(exePath), ConfigFileName)
	}
}

type MessageSender interface {
	Show(message string) // 有可能阻塞显示
	Close()
}

const (
	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_SYSTEMMODAL     = 0x00001000
	WM_CLOSE           = 0x0010
)

type MessageBoxSender struct {
	user32      *syscall.LazyDLL
	msgBox      *syscall.LazyProc
	findWindow  *syscall.LazyProc
	sendMessage *syscall.LazyProc
	title       *uint16
}

func NewMessageBoxSender(title string) *MessageBoxSender {
	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")
	findWindow := user32.NewProc("FindWindowW")
	sendMessage := user32.NewProc("SendMessageW")
	titleU16, _ := syscall.UTF16PtrFromString(title)

	return &MessageBoxSender{
		user32:      user32,
		msgBox:      msgBox,
		findWindow:  findWindow,
		sendMessage: sendMessage,
		title:       titleU16,
	}
}

func (s *MessageBoxSender) Show(message string) {
	messageU16, _ := syscall.UTF16PtrFromString(message)
	s.msgBox.Call(0, uintptr(unsafe.Pointer(messageU16)), uintptr(unsafe.Pointer(s.title)), MB_OK|MB_ICONINFORMATION|MB_SYSTEMMODAL)
}

func (s *MessageBoxSender) Close() {
	hwnd, _, _ := s.findWindow.Call(0, uintptr(unsafe.Pointer(s.title)))
	if hwnd != 0 {
		s.sendMessage.Call(hwnd, WM_CLOSE, 0, 0)
	}
}

type NotificationSender struct {
	title string
}

func NewNotificationSender(title string) *NotificationSender {
	return &NotificationSender{
		title: title,
	}
}

func (s *NotificationSender) Show(message string) {
	notification := toast.Notification{
		AppID:   s.title,
		Title:   s.title,
		Message: message,
		Actions: []toast.Action{
			{"protocol", "确定", ""},
		},
	}
	err := notification.Push()
	if err != nil {
		log.Println("Error showing reminder:", err)
	}
}

func (s *NotificationSender) Close() {
}
