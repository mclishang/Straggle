package tests

import (
	"testing"
	"time"

	"Straggle/internal/humanize"
	"Straggle/internal/i18n"
)

func TestDuration(t *testing.T) {
	cases := []struct {
		lang i18n.Lang
		in   time.Duration
		want string
	}{
		{i18n.EN, 30 * time.Second, "under a minute"},
		{i18n.EN, time.Minute, "1 minute"},
		{i18n.EN, 12 * time.Minute, "12 minutes"},
		{i18n.EN, 59 * time.Minute, "59 minutes"},
		{i18n.EN, time.Hour, "1 hour"},
		{i18n.EN, 3 * time.Hour, "3 hours"},
		{i18n.EN, 3*time.Hour + 5*time.Minute, "3 hours 5 minutes"},
		{i18n.EN, 25 * time.Hour, "1 day"},
		{i18n.EN, 48 * time.Hour, "2 days"},
		{i18n.ZH, 30 * time.Second, "不到 1 分钟"},
		{i18n.ZH, 12 * time.Minute, "12 分钟"},
		{i18n.ZH, 3 * time.Hour, "3 小时"},
		{i18n.ZH, 3*time.Hour + 5*time.Minute, "3 小时 5 分钟"},
		{i18n.ZH, 48 * time.Hour, "2 天"},
	}
	for _, c := range cases {
		i18n.Set(c.lang)
		if got := humanize.Duration(c.in); got != c.want {
			t.Errorf("[%s] Duration(%v) = %q, want %q", c.lang, c.in, got, c.want)
		}
	}
}

func TestYesterdayAcrossCalendarBoundaries(t *testing.T) {
	for _, now := range []time.Time{
		time.Date(2026, 10, 1, 0, 30, 0, 0, time.Local),
		time.Date(2026, 1, 1, 0, 30, 0, 0, time.Local),
		time.Date(2024, 3, 1, 0, 30, 0, 0, time.Local),
	} {
		for _, lang := range []i18n.Lang{i18n.EN, i18n.ZH} {
			i18n.Set(lang)
			before := now.AddDate(0, 0, -1)
			want := i18n.T("time.yesterday", "00:30")
			if got := humanize.StartTime(before, now); got != want {
				t.Errorf("StartTime(%v, %v) = %q, want %q", before, now, got, want)
			}
		}
	}
}

func TestStartTime(t *testing.T) {
	now := time.Date(2026, 10, 9, 20, 0, 0, 0, time.Local)
	cases := []struct {
		lang i18n.Lang
		in   time.Time
		want string
	}{
		{i18n.EN, time.Date(2026, 10, 9, 14, 2, 0, 0, time.Local), "Today 14:02"},
		{i18n.EN, time.Date(2026, 10, 8, 9, 31, 0, 0, time.Local), "Yesterday 09:31"},
		{i18n.EN, time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local), "Oct 03 at 14:02"},
		{i18n.EN, time.Date(2025, 12, 1, 8, 0, 0, 0, time.Local), "Dec 01, 2025 at 08:00"},
		{i18n.ZH, time.Date(2026, 10, 9, 14, 2, 0, 0, time.Local), "今天 14:02"},
		{i18n.ZH, time.Date(2026, 10, 8, 9, 31, 0, 0, time.Local), "昨天 09:31"},
		{i18n.ZH, time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local), "10月03日 14:02"},
		{i18n.ZH, time.Date(2025, 12, 1, 8, 0, 0, 0, time.Local), "2025年12月01日 08:00"},
		{i18n.EN, time.Date(2026, 9, 30, 14, 2, 0, 0, time.Local), "Sep 30 at 14:02"},
	}
	for _, c := range cases {
		i18n.Set(c.lang)
		if got := humanize.StartTime(c.in, now); got != c.want {
			t.Errorf("[%s] StartTime(%v) = %q, want %q", c.lang, c.in, got, c.want)
		}
	}
}
