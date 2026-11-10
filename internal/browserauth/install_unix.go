//go:build !windows

package browserauth

func registerHost(manifestTarget) error   { return nil }
func unregisterHost(manifestTarget) error { return nil }
