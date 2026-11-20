//go:build !windows

package app

import "time"

func uptime() time.Duration { return 0 }
