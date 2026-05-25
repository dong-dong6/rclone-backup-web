package logging

import (
	"log"
	"strings"
	"sync/atomic"
)

type Level int32

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var currentLevel atomic.Int32

func init() {
	currentLevel.Store(int32(LevelInfo))
}

func ParseLevel(raw string) Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "trace":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "info", "":
		return LevelInfo
	default:
		return LevelInfo
	}
}

func SetLevel(level Level) {
	currentLevel.Store(int32(level))
}

func LevelEnabled(level Level) bool {
	return level >= Level(currentLevel.Load())
}

func IsDebug() bool {
	return Level(currentLevel.Load()) == LevelDebug
}

func Debugf(format string, args ...any) {
	if !LevelEnabled(LevelDebug) {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}
