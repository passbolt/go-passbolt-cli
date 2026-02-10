package util

import (
	"context"

	"github.com/spf13/viper"
)

func GetContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), viper.GetViper().GetDuration("timeout"))
}
