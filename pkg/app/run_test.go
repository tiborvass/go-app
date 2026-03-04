//go:build !wasm
// +build !wasm

package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestRunSSR(t *testing.T) {
	defer delete(routes.routes, "/run-ssr")
	Route("/run-ssr", NewZeroComponentFactory(&runOKCompo{}))

	fake := fakebrowser.NewWindow("http://localhost/run-ssr")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, RunConfig{
		BrowserWindow: newBrowserWindowFromFake(fake),
	})
	require.NoError(t, err)

	got := strings.TrimSpace(fake.HTML())

	outputPath := filepath.Join(t.TempDir(), "run-ssr.html")
	err = os.WriteFile(outputPath, []byte(got), 0644)
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "run_ssr.golden.html")
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err)

	require.Equal(t, strings.TrimSpace(string(expected)), got)
}
