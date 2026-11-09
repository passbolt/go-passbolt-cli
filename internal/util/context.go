package util

import (
	"context"

	"github.com/spf13/viper"
)

var InvocationContext = context.Background()

func GetContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(InvocationContext, viper.GetViper().GetDuration("timeout"))
}
