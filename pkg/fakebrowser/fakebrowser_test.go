package fakebrowser

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueObjectAndArray(t *testing.T) {
	o := ValueOf(map[string]any{"foo": "bar", "n": 3})
	require.Equal(t, "bar", o.Get("foo").String())
	require.Equal(t, 3, o.Get("n").Int())

	a := ValueOf([]any{"a", 2})
	require.Equal(t, 2, a.Length())
	require.Equal(t, "a", a.Index(0).String())
	require.Equal(t, 2, a.Index(1).Int())
}

func TestWindowHistoryAndStorage(t *testing.T) {
	w := NewWindow("http://localhost/")
	g := w.Global()

	g.Get("localStorage").Call("setItem", "k", "v")
	require.Equal(t, "v", g.Get("localStorage").Call("getItem", "k").String())

	u, _ := url.Parse("http://localhost/next")
	w.AddHistory(u)
	require.Equal(t, "/next", w.URL().Path)
}

func TestCreateElement(t *testing.T) {
	w := NewWindow("http://localhost/")
	doc := w.Global().Get("document")

	el := doc.Call("createElement", "div")
	el.Call("setAttribute", "id", "x")
	require.Equal(t, "x", el.Call("getAttribute", "id").String())
}
