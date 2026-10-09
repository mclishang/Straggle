// Package tray shows the notification-area icon and menu behind the
// "keep running in the tray" setting.
package tray

import (
	"sync"
	"time"

	"fyne.io/systray"
	"git.sr.ht/~jackmordaunt/go-toast/v2"
)

// Actions are the callbacks the menu triggers, supplied by the App.
type Actions struct {
	Show    func()
	Hide    func()
	Cleanup func()
	Quit    func()
}

// Labels are the menu texts. The App rebuilds them from the text catalogue, so
// the tray follows the UI language, including a change made at runtime.
type Labels struct {
	Show        string
	ShowHint    string
	Hide        string
	HideHint    string
	CleanupHint string
	Quit        string
	QuitHint    string
}

// Controller wraps systray. The tray runs in its own goroutine with its own
// message loop, independent of the Wails main loop.
type Controller struct {
	actions Actions
	icon    []byte

	mu      sync.Mutex
	started bool
	ok      bool
	labels  Labels
	ready   chan struct{}
	show    *systray.MenuItem
	hide    *systray.MenuItem
	cleanup *systray.MenuItem
	quit    *systray.MenuItem
}

// New builds a tray controller.
func New(a Actions, icon []byte) *Controller {
	return &Controller{actions: a, icon: icon, ready: make(chan struct{})}
}

// Start runs the tray and waits up to 3 seconds for it to come up. Call
// Available to find out whether it did.
func (c *Controller) Start() {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.mu.Unlock()

	go systray.Run(c.onReady, func() {})

	select {
	case <-c.ready:
		c.mu.Lock()
		c.ok = true
		c.mu.Unlock()
	case <-time.After(3 * time.Second):
	}
}

// Available reports whether the tray icon was created.
func (c *Controller) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ok
}

// SetTooltip sets the hover text of the tray icon itself.
func (c *Controller) SetTooltip(s string) {
	if c.Available() {
		systray.SetTooltip(s)
	}
}

// SetLabels applies the menu texts, before or after the tray has come up.
func (c *Controller) SetLabels(l Labels) {
	c.mu.Lock()
	c.labels = l
	c.mu.Unlock()
	c.applyLabels()
}

// SetCleanupLabel sets the cleanup item title, which carries the orphan count
// and therefore changes with every scan.
func (c *Controller) SetCleanupLabel(s string) {
	c.mu.Lock()
	item := c.cleanup
	c.mu.Unlock()
	if item != nil {
		item.SetTitle(s)
	}
}

// Notify submits a Windows notification and reports whether Windows accepted it.
func (c *Controller) Notify(title, message string) bool {
	if err := toast.SetAppData(toast.AppData{AppID: "Straggle"}); err != nil {
		return false
	}
	n := toast.Notification{AppID: "Straggle", Title: title, Body: message}
	return n.Push() == nil
}

// Stop removes the tray icon.
func (c *Controller) Stop() {
	if c.Available() {
		systray.Quit()
	}
}

func (c *Controller) applyLabels() {
	c.mu.Lock()
	l := c.labels
	show, hide, cleanup, quit := c.show, c.hide, c.cleanup, c.quit
	c.mu.Unlock()
	if show == nil {
		return
	}
	show.SetTitle(l.Show)
	show.SetTooltip(l.ShowHint)
	hide.SetTitle(l.Hide)
	hide.SetTooltip(l.HideHint)
	cleanup.SetTooltip(l.CleanupHint)
	quit.SetTitle(l.Quit)
	quit.SetTooltip(l.QuitHint)
}

func (c *Controller) onReady() {
	c.mu.Lock()
	l := c.labels
	c.mu.Unlock()

	systray.SetIcon(c.icon)
	systray.SetTooltip("Straggle")
	show := systray.AddMenuItem(l.Show, l.ShowHint)
	hide := systray.AddMenuItem(l.Hide, l.HideHint)
	systray.AddSeparator()
	cleanup := systray.AddMenuItem("", l.CleanupHint)
	systray.AddSeparator()
	quit := systray.AddMenuItem(l.Quit, l.QuitHint)

	c.mu.Lock()
	c.show, c.hide, c.cleanup, c.quit = show, hide, cleanup, quit
	c.mu.Unlock()
	close(c.ready)

	go func() {
		for {
			select {
			case <-show.ClickedCh:
				if c.actions.Show != nil {
					c.actions.Show()
				}
			case <-hide.ClickedCh:
				if c.actions.Hide != nil {
					c.actions.Hide()
				}
			case <-cleanup.ClickedCh:
				if c.actions.Cleanup != nil {
					c.actions.Cleanup()
				}
			case <-quit.ClickedCh:
				if c.actions.Quit != nil {
					c.actions.Quit()
				}
				systray.Quit()
				return
			}
		}
	}()
}
