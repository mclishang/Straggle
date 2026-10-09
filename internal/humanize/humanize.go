// Package humanize renders durations and timestamps as interface text in the
// selected language.
package humanize

import (
	"time"

	"Straggle/internal/i18n"
)

// Duration renders a running time: "under a minute" / "12 minutes" /
// "3 hours 5 minutes" / "2 days".
func Duration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return i18n.T("time.underMinute")
	case d < time.Hour:
		m := int(d.Minutes())
		return i18n.Tn("time.minuteOne", "time.minutes", m, m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return i18n.Tn("time.hourOne", "time.hours", h, h)
		}
		return i18n.T("time.hoursMinutes",
			i18n.Tn("time.hourOne", "time.hours", h, h),
			i18n.Tn("time.minuteOne", "time.minutes", m, m))
	default:
		days := int(d.Hours() / 24)
		if days < 1 {
			days = 1
		}
		return i18n.Tn("time.dayOne", "time.days", days, days)
	}
}

// StartTime renders the moment a process started: "Today 14:02" /
// "Yesterday 09:31" / "Oct 03 at 14:02" / "Dec 01, 2025 at 08:00".
func StartTime(t, now time.Time) string {
	t = t.Local()
	now = now.Local()
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	yy, ym, yd := now.AddDate(0, 0, -1).Date()
	clock := t.Format("15:04")
	switch {
	case ty == ny && tm == nm && td == nd:
		return i18n.T("time.today", clock)
	case ty == yy && tm == ym && td == yd:
		return i18n.T("time.yesterday", clock)
	case ty == ny:
		return i18n.T("time.stamped", t.Format(i18n.T("time.layoutDay")), clock)
	default:
		return i18n.T("time.stamped", t.Format(i18n.T("time.layoutDate")), clock)
	}
}
