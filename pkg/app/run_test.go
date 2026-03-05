//go:build !wasm
// +build !wasm

package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type runOKCompo struct {
	Compo
}

func (c *runOKCompo) Render() UI {
	return Div().Text("ok")
}

type runSSRCompo struct {
	Compo
}

func (c *runSSRCompo) OnMount(ctx Context) {
	doc := Window().Get("document")

	title := doc.Call("createElement", "title")
	titleText := doc.Call("createTextNode", "SSR Window Title")
	title.Call("appendChild", titleText)
	doc.Get("head").Call("appendChild", title)

	dynamic := doc.Call("createElement", "section")
	dynamic.Call("setAttribute", "id", "dynamic")
	dynamic.Call("setAttribute", "data-origin", "window-api")
	dynamic.Call("setAttribute", "data-stage", "0")
	dynamic.Call("appendChild", doc.Call("createTextNode", "dynamic content"))
	doc.Get("body").Call("appendChild", dynamic)

	Window().ScrollToID("dynamic")

	c.after(ctx, 30*time.Millisecond, func() {
		doc := Window().Get("document")
		titleText.Set("nodeValue", "SSR Window Title (phase one)")
		dynamic.Call("setAttribute", "data-stage", "1")

		step := doc.Call("createElement", "p")
		step.Call("setAttribute", "id", "dynamic-step-1")
		step.Call("appendChild", doc.Call("createTextNode", "phase one"))
		dynamic.Call("appendChild", step)
	})

	c.after(ctx, 70*time.Millisecond, func() {
		doc := Window().Get("document")
		titleText.Set("nodeValue", "SSR Window Title (settled)")
		dynamic.Call("setAttribute", "data-stage", "2")

		step := doc.Call("createElement", "aside")
		step.Call("setAttribute", "id", "dynamic-step-2")
		step.Call("setAttribute", "data-final", "true")
		step.Call("appendChild", doc.Call("createTextNode", "phase two"))
		dynamic.Call("appendChild", step)

		Window().ScrollToID("dynamic-step-2")
	})
}

func (c *runSSRCompo) after(ctx Context, delay time.Duration, fn func()) {
	ctx.Async(func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}

		if ctx.Err() != nil {
			return
		}
		fn()
	})
}

func (c *runSSRCompo) Render() UI {
	return Div().
		ID("ssr-root").
		Body(
			H1().Text("SSR test"),
			P().Text("window calls"),
			Ul().Body(
				Li().Text("alpha"),
				Li().Text("beta"),
			),
		)
}

func TestRunWithoutRuntime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, RunConfig{})
	require.NoError(t, err)
}

func TestRunWithFakeBrowser(t *testing.T) {
	defer delete(routes.routes, "/run-ok")
	Route("/run-ok", NewZeroComponentFactory(&runOKCompo{}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	w := newBrowserWindow("http://localhost/run-ok")
	err := Run(ctx, RunConfig{
		BrowserWindow: w,
	})
	require.NoError(t, err)
}

func TestRunWhenOnBrowserNoopOnServer(t *testing.T) {
	RunWhenOnBrowser()
}

func TestRunSSR(t *testing.T) {
	defer delete(routes.routes, "/run-ssr")
	Route("/run-ssr", NewZeroComponentFactory(&runSSRCompo{}))

	shortHTML := runSSRHTMLWithMaxIdle(t, 10*time.Millisecond)
	longHTML := runSSRHTMLWithMaxIdle(t, 160*time.Millisecond)

	require.NotEqual(t, shortHTML, longHTML)
	require.NotContains(t, shortHTML, `id="dynamic-step-1"`)
	require.NotContains(t, shortHTML, `id="dynamic-step-2"`)
	require.Contains(t, longHTML, `id="dynamic-step-1"`)
	require.Contains(t, longHTML, `id="dynamic-step-2"`)

	shortGoldenPath := filepath.Join("testdata", "run_ssr_short.golden.html")
	shortExpected, err := os.ReadFile(shortGoldenPath)
	require.NoError(t, err)

	longGoldenPath := filepath.Join("testdata", "run_ssr.golden.html")
	longExpected, err := os.ReadFile(longGoldenPath)
	require.NoError(t, err)

	require.Equal(t, string(shortExpected), shortHTML)
	require.Equal(t, string(longExpected), longHTML)
}

func runSSRHTMLWithMaxIdle(t *testing.T, maxIdleDuration time.Duration) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	w := newBrowserWindow("http://localhost/run-ssr")
	err := Run(ctx, RunConfig{
		BrowserWindow:   w,
		MaxIdleDuration: maxIdleDuration,
	})
	require.NoError(t, err)
	require.NoError(t, ctx.Err())

	return w.HTML()
}
