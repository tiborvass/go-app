package app

import "sync"

var (
	runtimeModeMutex sync.Mutex
	runtimeIsClient  = IsClient
)

func isClientRuntime() bool {
	return runtimeIsClient
}

func isServerRuntime() bool {
	return !runtimeIsClient
}

func withRuntimeOverrides(client bool, w BrowserWindow) (restore func()) {
	runtimeModeMutex.Lock()

	previousWindow := window
	previousClientMode := runtimeIsClient

	if w != nil {
		window = w
	}
	runtimeIsClient = client

	return func() {
		runtimeIsClient = previousClientMode
		window = previousWindow
		runtimeModeMutex.Unlock()
	}
}
