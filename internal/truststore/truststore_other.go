//go:build !linux && !darwin && !windows

package truststore

import "context"

func installPlatform(_ context.Context, _ string) (Result, error) {
	return Result{}, ErrUnsupportedPlatform
}

func uninstallPlatform(_ context.Context, _ string) (Result, error) {
	return Result{}, ErrUnsupportedPlatform
}
