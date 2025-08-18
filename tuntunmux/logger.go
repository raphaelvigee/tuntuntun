package tuntunmux

import (
	"context"
	"fmt"
	"log/slog"
)

type logger struct {
	logger *slog.Logger
	ctx    context.Context
}

func (l logger) Print(v ...interface{}) {
	l.logger.Log(l.ctx, slog.LevelError, fmt.Sprint(v...))
}

func (l logger) Printf(format string, v ...interface{}) {
	l.logger.Log(l.ctx, slog.LevelError, fmt.Sprintf(format, v...))
}

func (l logger) Println(v ...interface{}) {
	l.Print(v...)
}
