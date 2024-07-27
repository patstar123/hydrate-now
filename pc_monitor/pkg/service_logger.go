package pkg

import (
	"fmt"
	"github.com/kardianos/service"
	"github.com/livekit/protocol/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ServiceLogger struct {
	impl  service.Logger
	level zap.AtomicLevel
}

func NewServiceLogger(sl service.Logger, level string) *ServiceLogger {
	return &ServiceLogger{
		impl:  sl,
		level: zap.NewAtomicLevelAt(logger.ParseZapLevel(level)),
	}
}

func (l *ServiceLogger) isEnabled(level zapcore.Level) bool {
	return level >= l.level.Level()
}

func (l *ServiceLogger) IsEnabledDebug() bool {
	return l.isEnabled(zapcore.DebugLevel)
}

func (l *ServiceLogger) WithValues(keysAndValues ...interface{}) logger.Logger {
	return l
}

func (l *ServiceLogger) WithName(name string) logger.Logger {
	return l
}

func (l *ServiceLogger) WithComponent(component string) logger.Logger {
	return l
}

func (l *ServiceLogger) WithCallDepth(depth int) logger.Logger {
	return l
}

func (l *ServiceLogger) WithItemSampler() logger.Logger {
	return l
}

func (l *ServiceLogger) WithoutSampler() logger.Logger {
	return l
}

func (l *ServiceLogger) SetLevel(level string) error {
	lvl := logger.ParseZapLevel(level)
	l.level = zap.NewAtomicLevelAt(lvl)
	return nil
}

func (l *ServiceLogger) GetLevel() string {
	return l.level.String()
}

func (l *ServiceLogger) Debug(args ...interface{}) {
	fmt.Print(args...)
}

func (l *ServiceLogger) Info(args ...interface{}) {
	l.impl.Info(args...)
}

func (l *ServiceLogger) Warn(args ...interface{}) {
	l.impl.Warning(args...)
}

func (l *ServiceLogger) Error(args ...interface{}) {
	l.impl.Error(args...)
}

func (l *ServiceLogger) Debugf(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}

func (l *ServiceLogger) Infof(template string, args ...interface{}) {
	l.impl.Infof(template, args...)
}

func (l *ServiceLogger) Warnf(err error, template string, args ...interface{}) {
	if err != nil {
		l.impl.Warningf(template+"(error: %v)", append(args, err)...)
	} else {
		l.impl.Warningf(template, args...)
	}
}

func (l *ServiceLogger) Errorf(err error, template string, args ...interface{}) {
	if err != nil {
		l.impl.Errorf(template+"(error: %v)", append(args, err)...)
	} else {
		l.impl.Errorf(template, args...)
	}
}

func (l *ServiceLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.Debug(msg, keysAndValues)
}

func (l *ServiceLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.Info(msg, keysAndValues)
}

func (l *ServiceLogger) Warnw(msg string, err error, keysAndValues ...interface{}) {
	l.Warn(msg, append(keysAndValues, err))
}

func (l *ServiceLogger) Errorw(msg string, err error, keysAndValues ...interface{}) {
	l.Error(msg, append(keysAndValues, err))
}

func (l *ServiceLogger) Debugln(args ...interface{}) {
	l.Debug(args...)
}

func (l *ServiceLogger) Infoln(args ...interface{}) {
	l.Info(args...)
}

func (l *ServiceLogger) Warnln(args ...interface{}) {
	l.Warn(args...)
}

func (l *ServiceLogger) Errorln(args ...interface{}) {
	l.Error(args...)
}
