//go:build !wasm
// +build !wasm

package app

import (
	"net/url"

	"github.com/maxence-charriere/go-app/v10/pkg/fakebrowser"
)

type value struct {
	fb fakebrowser.Value
}

func (v value) Bool() bool { return v.fb.Bool() }

func (v value) Call(m string, args ...any) Value {
	return value{fb: v.fb.Call(m, fakeBrowserArgs(args)...)}
}

func (v value) Delete(p string) { v.fb.Delete(p) }

func (v value) Equal(w Value) bool {
	return v.fb.Equal(fakeBrowserValueOf(w))
}

func (v value) Float() float64 { return v.fb.Float() }

func (v value) Get(p string) Value {
	return value{fb: v.fb.Get(p)}
}

func (v value) Index(i int) Value {
	return value{fb: v.fb.Index(i)}
}

func (v value) InstanceOf(t Value) bool {
	return v.fb.InstanceOf(fakeBrowserValueOf(t))
}

func (v value) Int() int { return v.fb.Int() }

func (v value) Invoke(args ...any) Value {
	return value{fb: v.fb.Invoke(fakeBrowserArgs(args)...)}
}

func (v value) IsNaN() bool       { return v.fb.IsNaN() }
func (v value) IsNull() bool      { return v.fb.IsNull() }
func (v value) IsUndefined() bool { return v.fb.IsUndefined() }

func (v value) JSValue() Value { return v }

func (v value) Length() int { return v.fb.Length() }

func (v value) New(args ...any) Value {
	return value{fb: v.fb.New(fakeBrowserArgs(args)...)}
}

func (v value) Set(p string, x any) {
	v.fb.Set(p, fakeBrowserValueOfAny(x))
}

func (v value) SetIndex(i int, x any) {
	v.fb.SetIndex(i, fakeBrowserValueOfAny(x))
}

func (v value) String() string { return v.fb.String() }
func (v value) Truthy() bool   { return v.fb.Truthy() }

func (v value) Type() Type {
	return Type(v.fb.Type())
}

func (v value) Then(f func(Value)) {
	v.fb.Then(func(v fakebrowser.Value) {
		f(value{fb: v})
	})
}

func (v value) getAttr(k string) string {
	return v.Call("getAttribute", k).String()
}

func (v value) setAttr(k, val string) { v.Call("setAttribute", k, val) }
func (v value) delAttr(k string)      { v.Call("removeAttribute", k) }

func (v value) firstChild() Value {
	return v.Get("firstChild")
}

func (v value) appendChild(c Wrapper) {
	v.Call("appendChild", c)
}

func (v value) replaceChild(new, old Wrapper) {
	v.Call("replaceChild", new, old)
}

func (v value) removeChild(c Wrapper) {
	v.Call("removeChild", c)
}

func (v value) firstElementChild() Value {
	return v.Get("firstElementChild")
}

func (v value) addEventListener(event string, fn Func, options map[string]any) {
	if len(options) == 0 {
		v.Call("addEventListener", event, fn)
		return
	}
	v.Call("addEventListener", event, fn, options)
}

func (v value) removeEventListener(event string, fn Func) {
	v.Call("removeEventListener", event, fn)
}

func (v value) setNodeValue(val string) { v.Set("nodeValue", val) }
func (v value) setInnerHTML(val string) { v.Set("innerHTML", val) }
func (v value) setInnerText(val string) { v.Set("innerText", val) }

func null() Value      { return value{fb: fakebrowser.Null()} }
func undefined() Value { return value{fb: fakebrowser.Undefined()} }

func valueOf(x any) Value {
	return value{fb: fakeBrowserValueOfAny(x)}
}

type function struct {
	value
}

func (f function) Release() {}

func funcOf(fn func(this Value, args []Value) any) Func {
	return function{
		value: value{
			fb: fakebrowser.FuncOf(func(this fakebrowser.Value, args []fakebrowser.Value) fakebrowser.Value {
				in := make([]Value, len(args))
				for i, arg := range args {
					in[i] = value{fb: arg}
				}
				return fakeBrowserValueOfAny(fn(value{fb: this}, in))
			}),
		},
	}
}

type browserWindow struct {
	value
	fake *fakebrowser.Window
	body UI
}

func newBrowserWindow(url string) *browserWindow {
	fb := fakebrowser.NewWindow(url)
	return &browserWindow{
		value: value{fb: fb.Global()},
		fake:  fb,
	}
}

func (w *browserWindow) HTML() string {
	return w.fake.HTML()
}

func (w browserWindow) URL() *url.URL {
	return w.fake.URL()
}

func (w browserWindow) Size() (width, height int) {
	return w.fake.Size()
}

func (w browserWindow) CursorPosition() (x, y int) {
	return w.fake.CursorPosition()
}

func (w browserWindow) setCursorPosition(x, y int) {
	w.fake.SetCursorPosition(x, y)
}

func (w *browserWindow) GetElementByID(id string) Value {
	return value{fb: w.fake.GetElementByID(id)}
}

func (w *browserWindow) ScrollToID(id string) {
	w.fake.ScrollToID(id)
}

func (w *browserWindow) setBody(body UI) {
	w.body = body
}

func (w *browserWindow) createElement(tag, xmlns string) (Value, error) {
	v, err := w.fake.CreateElement(tag, xmlns)
	if err != nil {
		return nil, err
	}
	return value{fb: v}, nil
}

func (w *browserWindow) createTextNode(v string) Value {
	return value{fb: w.fake.CreateTextNode(v)}
}

func (w *browserWindow) addHistory(u *url.URL) {
	w.fake.AddHistory(u)
}

func (w *browserWindow) replaceHistory(u *url.URL) {
	w.fake.ReplaceHistory(u)
}

func copyBytesToGo(dst []byte, src Value) int {
	return fakebrowser.CopyBytesToGo(dst, fakeBrowserValueOf(src))
}

func copyBytesToJS(dst Value, src []byte) int {
	return fakebrowser.CopyBytesToJS(fakeBrowserValueOf(dst), src)
}

func fakeBrowserArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	for i, arg := range args {
		out[i] = fakeBrowserValueOfAny(arg)
	}
	return out
}

func fakeBrowserValueOf(v Value) fakebrowser.Value {
	switch x := v.(type) {
	case value:
		return x.fb
	case Wrapper:
		return fakeBrowserValueOf(x.JSValue())
	default:
		return fakebrowser.ValueOf(nil)
	}
}

func fakeBrowserValueOfAny(v any) fakebrowser.Value {
	switch x := v.(type) {
	case nil:
		return fakebrowser.Null()
	case value:
		return x.fb
	case Wrapper:
		return fakeBrowserValueOfAny(x.JSValue())
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, val := range x {
			m[k] = fakeBrowserValueOfAny(val)
		}
		return fakebrowser.ValueOf(m)
	case []any:
		s := make([]any, len(x))
		for i, val := range x {
			s[i] = fakeBrowserValueOfAny(val)
		}
		return fakebrowser.ValueOf(s)
	default:
		return fakebrowser.ValueOf(x)
	}
}
