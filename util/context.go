package util

import (
	"context"

	"github.com/spf13/viper"
)

func GetContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetViper().GetDuration("timeout"))
	_ = cancel
	return ctx
}
