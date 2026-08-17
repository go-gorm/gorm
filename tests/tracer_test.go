package tests_test

import (
	"context"
	"time"

	"gorm.io/gorm/logger"
)

type Tracer struct {
	Logger logger.Interface
	Test   func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

func (t Tracer) LogMode(level logger.LogLevel) logger.Interface {
	return t.Logger.LogMode(level)
}

func (t Tracer) Info(ctx context.Context, s string, i ...interface{}) {
	t.Logger.Info(ctx, s, i...)
}

func (t Tracer) Warn(ctx context.Context, s string, i ...interface{}) {
	t.Logger.Warn(ctx, s, i...)
}

func (t Tracer) Error(ctx context.Context, s string, i ...interface{}) {
	t.Logger.Error(ctx, s, i...)
}

func (t Tracer) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	t.Logger.Trace(ctx, begin, fc, err)
	t.Test(ctx, begin, fc, err)
}
