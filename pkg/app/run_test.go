//go:build !wasm
// +build !wasm

package app

import (
	"context"
	"testing"
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/fakebrowser"
	"github.com/stretchr/testify/require"
)

type runOKCompo struct {
	Compo
}

func (c *runOKCompo) Render() UI {
	return Div().Text("ok")
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

	fake := fakebrowser.NewWindow("http://localhost/run-ok")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, RunConfig{
		BrowserWindow: newBrowserWindowFromFake(fake),
	})
	require.NoError(t, err)
}

func TestRunWhenOnBrowserNoopOnServer(t *testing.T) {
	RunWhenOnBrowser()
}
