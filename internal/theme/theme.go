// Package theme reports the Windows light/dark preference: read once for the
// current value, poll for changes.
package theme

import (
	"context"
	"time"

	"golang.org/x/sys/windows/registry"
)

const personalizeKey = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

// IsDark reads Windows "Personalization → Colors → app mode", where
// AppsUseLightTheme = 0 means dark.
func IsDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, personalizeKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false
	}
	return v == 0
}

// Watch calls onChange with the current theme right away, then polls every
// interval and calls it again on every change.
func Watch(ctx context.Context, interval time.Duration, onChange func(dark bool)) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	last := IsDark()
	onChange(last)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cur := IsDark()
			if cur != last {
				last = cur
				onChange(cur)
			}
		}
	}
}
