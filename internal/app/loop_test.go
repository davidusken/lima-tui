package app

import (
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/davidusken/lima-tui/internal/lima"
)

// Tests here run the real tview event loop on a simulated screen, for the
// code that talks to limactl in the background and reports back through
// QueueUpdateDraw.

// startLoop runs a's event loop until the test ends. Set a's limactl fakes
// before calling it. The returned channel is closed once the loop stops.
func startLoop(t *testing.T, a *app) <-chan struct{} {
	t.Helper()
	a.tv.SetScreen(tcell.NewSimulationScreen("UTF-8"))
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		if err := a.tv.SetRoot(a.pages, true).Run(); err != nil {
			t.Errorf("event loop: %v", err)
		}
	}()
	t.Cleanup(func() {
		a.tv.Stop()
		<-stopped
	})
	return stopped
}

// onLoop runs f on the event loop and waits for it to finish.
func onLoop(a *app, f func()) {
	done := make(chan struct{})
	a.tv.QueueUpdate(func() {
		defer close(done)
		f()
	})
	<-done
}

// eventually checks cond on the event loop until it holds.
func eventually(t *testing.T, a *app, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		var ok bool
		onLoop(a, func() { ok = cond() })
		if ok {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

// waitFor fails the test if ch doesn't receive within a few seconds.
func waitFor[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

func keyEvent(code tcell.Key) *tcell.EventKey { return tcell.NewEventKey(code, 0, tcell.ModNone) }

func TestRunLimactlBusyUntilDone(t *testing.T) {
	a := newTestApp()
	release := make(chan error)
	gotArgs := make(chan []string, 1)
	a.runCommand = func(args ...string) error {
		gotArgs <- args
		return <-release
	}
	var lists atomic.Int32
	a.listVMs = func() ([]lima.Instance, error) {
		lists.Add(1)
		return []lima.Instance{{Name: "vm1", Status: lima.StatusRunning}}, nil
	}
	startLoop(t, a)

	onLoop(a, func() {
		a.vms = []lima.Instance{{Name: "vm1", Status: lima.StatusStopped}}
		a.runLimactl("vm1", actStarting, "Started", "start", "vm1")
	})
	if args := waitFor(t, gotArgs, "limactl to run"); !slices.Equal(args, []string{"start", "vm1"}) {
		t.Errorf("limactl args = %q, want [start vm1]", args)
	}
	onLoop(a, func() {
		if a.busy["vm1"] != actStarting {
			t.Errorf("busy[vm1] = %q while running, want %s", a.busy["vm1"], actStarting)
		}
		if got := a.table.GetCell(1, 1).Text; got != "Starting…" {
			t.Errorf("status cell = %q while running, want Starting…", got)
		}
	})

	release <- nil
	eventually(t, a, "the VM to be idle and the list reloaded", func() bool {
		return len(a.busy) == 0 && strings.Contains(a.message, "Started vm1") &&
			len(a.vms) == 1 && a.vms[0].Status == lima.StatusRunning
	})
	if lists.Load() == 0 {
		t.Error("the VM list wasn't reloaded after the action")
	}
}

func TestRunLimactlShowsError(t *testing.T) {
	a := newTestApp()
	a.runCommand = func(...string) error { return errors.New("limactl stop: [boom]") }
	a.listVMs = func() ([]lima.Instance, error) { return nil, nil }
	startLoop(t, a)

	onLoop(a, func() { a.runLimactl("vm1", actStopping, "Stopped", "stop", "vm1") })
	eventually(t, a, "the error in the status bar", func() bool {
		// Escaped, so the brackets don't read as a colour tag.
		return len(a.busy) == 0 && strings.Contains(a.message, "limactl stop: [boom[]")
	})
}

func TestLoadErrorShown(t *testing.T) {
	a := newTestApp()
	a.listVMs = func() ([]lima.Instance, error) { return nil, errors.New("limactl list: no answer") }
	startLoop(t, a)

	onLoop(a, a.manualRefresh)
	eventually(t, a, "the error in the status bar", func() bool {
		return !a.refreshing && strings.Contains(a.message, "no answer")
	})
}

// Pressing r while a reload is running must not report the reload that was
// already under way: it may have read the VM list before the key press.
func TestManualRefreshWaitsForItsOwnLoad(t *testing.T) {
	a := newTestApp()
	started := make(chan int, 10)
	release := make(chan []lima.Instance)
	var calls atomic.Int32
	a.listVMs = func() ([]lima.Instance, error) {
		started <- int(calls.Add(1))
		return <-release, nil
	}
	startLoop(t, a)

	onLoop(a, a.refresh) // a background reload
	waitFor(t, started, "the first load")
	onLoop(a, func() {
		a.manualRefresh()
		a.refresh() // more calls in the meantime queue just one more load
		a.refresh()
	})

	release <- vmsNamed("old")
	if n := waitFor(t, started, "the second load"); n != 2 {
		t.Fatalf("load #%d started, want #2", n)
	}
	onLoop(a, func() {
		if len(a.vms) != 1 || a.vms[0].Name != "old" {
			t.Fatalf("vms = %v, want the first load's", a.vms)
		}
		if !a.refreshing || strings.Contains(a.message, "Refreshed") {
			t.Errorf("refreshing=%v message=%q after the older load; want still refreshing",
				a.refreshing, a.message)
		}
	})

	release <- vmsNamed("new")
	eventually(t, a, "Refreshed with the new list", func() bool {
		return !a.refreshing && strings.Contains(a.message, "Refreshed") &&
			len(a.vms) == 1 && a.vms[0].Name == "new"
	})
	select {
	case n := <-started:
		t.Errorf("load #%d started; queued refreshes should run once", n)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestQuitWhileIdle(t *testing.T) {
	a := newTestApp()
	stopped := startLoop(t, a)
	onLoop(a, a.quit)
	waitFor(t, stopped, "the app to stop")
}

func TestQuitAsksWhileBusy(t *testing.T) {
	a := newTestApp()
	stopped := startLoop(t, a)
	onLoop(a, func() {
		a.busy["web"] = actStopping
		a.busy["db"] = actStarting
		a.quit()
		if !a.pages.HasPage(pageConfirm) {
			t.Fatal("quit with actions running didn't ask")
		}
	})

	// Cancel has the focus, so Enter keeps the app running.
	a.tv.QueueEvent(keyEvent(tcell.KeyEnter))
	eventually(t, a, "the dialog to close", func() bool { return !a.pages.HasPage(pageConfirm) })
	select {
	case <-stopped:
		t.Fatal("Enter on the quit dialog quit; Cancel should be the default")
	default:
	}

	onLoop(a, a.quit)
	a.tv.QueueEvent(keyEvent(tcell.KeyTab)) // to Quit
	a.tv.QueueEvent(keyEvent(tcell.KeyEnter))
	waitFor(t, stopped, "the app to stop")
}

func TestQuitDialogListsActions(t *testing.T) {
	a := newTestApp()
	a.busy["web"] = actStopping
	a.busy["db"] = actStarting
	a.quit()

	_, page := a.pages.GetFrontPage()
	if page == nil {
		t.Fatal("no dialog")
	}
	// The modal's text isn't exposed, so draw it and read the screen.
	scr := tcell.NewSimulationScreen("UTF-8")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(100, 30)
	page.SetRect(0, 0, 100, 30)
	page.Draw(scr)
	scr.Show()
	cells, w, _ := scr.GetContents()
	var text strings.Builder
	for i, c := range cells {
		if i%w == 0 {
			text.WriteByte('\n')
		}
		text.WriteString(string(c.Runes))
	}
	for _, want := range []string{"VM actions are still running", "Starting db", "Stopping web", "They keep"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("dialog doesn't say %q:%s", want, text.String())
		}
	}
}
