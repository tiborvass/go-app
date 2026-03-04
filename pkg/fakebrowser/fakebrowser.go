package fakebrowser

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
)

type Value struct {
	v any
}

type function func(this Value, args []Value) Value

type object struct {
	props   map[string]Value
	arr     []Value
	fn      function
	ctor    function
	promise *promise
	node    *node
}

type promise struct {
	resolved bool
	value    Value
	then     []Value
}

type node struct {
	tag       string
	nodeValue string
	innerHTML string
	innerText string
	attrs     map[string]string
	parent    *object
	children  []*object
	listeners map[string][]Value
	object    *object
}

type storage struct {
	items map[string]string
}

type Window struct {
	mutex          sync.Mutex
	rawURL         *url.URL
	width          int
	height         int
	cursorX        int
	cursorY        int
	body           *node
	head           *node
	byID           map[string]*node
	localStorage   *storage
	sessionStorage *storage
	global         *object
	document       *object
	location       *object
	history        *object
	console        *object
}

var (
	undefinedSentinel = struct{}{}
	nullSentinel      = struct{}{}
)

func Undefined() Value { return Value{v: undefinedSentinel} }
func Null() Value      { return Value{v: nullSentinel} }

func ValueOf(v any) Value {
	switch x := v.(type) {
	case Value:
		return x
	case nil:
		return Null()
	case bool:
		return Value{v: x}
	case int:
		return Value{v: float64(x)}
	case int8:
		return Value{v: float64(x)}
	case int16:
		return Value{v: float64(x)}
	case int32:
		return Value{v: float64(x)}
	case int64:
		return Value{v: float64(x)}
	case uint:
		return Value{v: float64(x)}
	case uint8:
		return Value{v: float64(x)}
	case uint16:
		return Value{v: float64(x)}
	case uint32:
		return Value{v: float64(x)}
	case uint64:
		return Value{v: float64(x)}
	case float32:
		return Value{v: float64(x)}
	case float64:
		return Value{v: x}
	case string:
		return Value{v: x}
	case []any:
		a := &object{arr: make([]Value, len(x)), props: map[string]Value{}}
		for i, item := range x {
			a.arr[i] = ValueOf(item)
		}
		return Value{v: a}
	case map[string]any:
		o := &object{props: map[string]Value{}}
		for k, item := range x {
			o.props[k] = ValueOf(item)
		}
		return Value{v: o}
	case []byte:
		a := make([]Value, len(x))
		for i, b := range x {
			a[i] = Value{v: float64(b)}
		}
		return Value{v: &object{arr: a, props: map[string]Value{}}}
	default:
		return Value{v: x}
	}
}

func FuncOf(fn func(this Value, args []Value) Value) Value {
	return Value{v: &object{props: map[string]Value{}, fn: function(fn)}}
}

func NewResolvedPromise(v Value) Value {
	p := &promise{resolved: true, value: v}
	return Value{v: &object{props: map[string]Value{}, promise: p}}
}

func NewWindow(raw string) *Window {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		u, _ = url.Parse("http://localhost/")
	}

	w := &Window{
		rawURL:         u,
		width:          1280,
		height:         720,
		byID:           map[string]*node{},
		localStorage:   &storage{items: map[string]string{}},
		sessionStorage: &storage{items: map[string]string{}},
	}

	w.body = newNode("body")
	w.head = newNode("head")

	// Ensure body has a first element child. The app runtime expects this.
	w.body.appendChild(newNode("div"))

	w.document = w.newDocument()
	w.location = w.newLocation()
	w.history = w.newHistory()
	w.console = w.newConsole()
	w.global = w.newWindowObject()

	return w
}

func (w *Window) Global() Value {
	return Value{v: w.global}
}

func (w *Window) URL() *url.URL {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	u := *w.rawURL
	return &u
}

func (w *Window) SetURL(u *url.URL) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	copy := *u
	w.rawURL = &copy
	if w.location != nil {
		w.location.props["href"] = ValueOf(copy.String())
	}
}

func (w *Window) Size() (int, int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.width, w.height
}

func (w *Window) SetSize(width, height int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.width, w.height = width, height
}

func (w *Window) CursorPosition() (int, int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.cursorX, w.cursorY
}

func (w *Window) SetCursorPosition(x, y int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.cursorX, w.cursorY = x, y
}

func (w *Window) GetElementByID(id string) Value {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	n, ok := w.byID[id]
	if !ok {
		return Null()
	}
	return Value{v: n.object}
}

func (w *Window) ScrollToID(string) {}

func (w *Window) CreateElement(tag, xmlns string) (Value, error) {
	_ = xmlns
	n := newNode(tag)
	return Value{v: n.object}, nil
}

func (w *Window) CreateTextNode(v string) Value {
	n := newNode("#text")
	n.nodeValue = v
	n.object.props["nodeValue"] = ValueOf(v)
	return Value{v: n.object}
}

func (w *Window) AddHistory(u *url.URL) {
	w.SetURL(u)
}

func (w *Window) ReplaceHistory(u *url.URL) {
	w.SetURL(u)
}

// HTML returns a serialized HTML snapshot of the fake document.
func (w *Window) HTML() string {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString("<html>")
	b.WriteString("<head>")
	for _, child := range w.head.children {
		renderNode(&b, child.node)
	}
	b.WriteString("</head>")
	b.WriteString("<body>")
	for _, child := range w.body.children {
		renderNode(&b, child.node)
	}
	b.WriteString("</body>")
	b.WriteString("</html>")
	return b.String()
}

func (w *Window) newWindowObject() *object {
	o := &object{props: map[string]Value{}}
	o.props["document"] = Value{v: w.document}
	o.props["history"] = Value{v: w.history}
	o.props["location"] = Value{v: w.location}
	o.props["localStorage"] = Value{v: w.newStorageObject(w.localStorage)}
	o.props["sessionStorage"] = Value{v: w.newStorageObject(w.sessionStorage)}
	o.props["console"] = Value{v: w.console}
	o.props["innerWidth"] = ValueOf(w.width)
	o.props["innerHeight"] = ValueOf(w.height)

	o.props["open"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) > 0 {
			raw := args[0].String()
			if u, err := url.Parse(raw); err == nil {
				w.SetURL(u)
			}
		}
		return Undefined()
	})

	o.props["Promise"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) == 0 {
			return NewResolvedPromise(Undefined())
		}
		cb := args[0]
		resolve := FuncOf(func(_ Value, resArgs []Value) Value {
			if len(resArgs) == 0 {
				return NewResolvedPromise(Undefined())
			}
			return NewResolvedPromise(resArgs[0])
		})
		cb.Invoke(resolve)
		return NewResolvedPromise(Undefined())
	})

	o.props["BroadcastChannel"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		_ = args
		ch := &object{props: map[string]Value{}}
		ch.props["postMessage"] = FuncOf(func(Value, []Value) Value { return Undefined() })
		ch.props["addEventListener"] = FuncOf(func(Value, []Value) Value { return Undefined() })
		ch.props["removeEventListener"] = FuncOf(func(Value, []Value) Value { return Undefined() })
		ch.props["close"] = FuncOf(func(Value, []Value) Value { return Undefined() })
		return Value{v: ch}
	})

	o.props["Notification"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		n := &object{props: map[string]Value{}}
		if len(args) > 0 {
			n.props["title"] = args[0]
		}
		n.props["close"] = FuncOf(func(Value, []Value) Value { return Undefined() })
		return Value{v: n}
	})
	o.props["CustomEvent"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		e := &object{props: map[string]Value{}}
		if len(args) > 0 {
			e.props["type"] = args[0]
		}
		if len(args) > 1 {
			e.props["detail"] = args[1].Get("detail")
		}
		return Value{v: e}
	})

	o.props["matchMedia"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		_ = args
		mm := &object{props: map[string]Value{}}
		mm.props["matches"] = ValueOf(false)
		return Value{v: mm}
	})

	return o
}

func renderNode(b *strings.Builder, n *node) {
	if n == nil {
		return
	}

	tag := strings.ToLower(n.tag)
	if tag == "#text" {
		if nodeValue, ok := n.object.props["nodeValue"]; ok {
			b.WriteString(nodeValue.String())
			return
		}
		b.WriteString(n.nodeValue)
		return
	}

	b.WriteString("<")
	b.WriteString(tag)

	keys := make([]string, 0, len(n.attrs))
	for k := range n.attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString("=\"")
		b.WriteString(n.attrs[k])
		b.WriteString("\"")
	}
	b.WriteString(">")

	for _, child := range n.children {
		renderNode(b, child.node)
	}

	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
}

func (w *Window) newLocation() *object {
	o := &object{props: map[string]Value{}}
	o.props["href"] = ValueOf(w.rawURL.String())
	return o
}

func (w *Window) newHistory() *object {
	o := &object{props: map[string]Value{}}
	o.props["pushState"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) >= 3 {
			raw := args[2].String()
			if u, err := url.Parse(raw); err == nil {
				w.SetURL(u)
			}
		}
		return Undefined()
	})
	o.props["replaceState"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) >= 3 {
			raw := args[2].String()
			if u, err := url.Parse(raw); err == nil {
				w.SetURL(u)
			}
		}
		return Undefined()
	})
	return o
}

func (w *Window) newConsole() *object {
	o := &object{props: map[string]Value{}}
	o.props["log"] = FuncOf(func(this Value, args []Value) Value { _ = this; _ = args; return Undefined() })
	o.props["error"] = FuncOf(func(this Value, args []Value) Value { _ = this; _ = args; return Undefined() })
	return o
}

func (w *Window) newStorageObject(s *storage) *object {
	o := &object{props: map[string]Value{}}
	o.props["setItem"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) >= 2 {
			s.items[args[0].String()] = args[1].String()
		}
		return Undefined()
	})
	o.props["getItem"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) == 0 {
			return Null()
		}
		v, ok := s.items[args[0].String()]
		if !ok {
			return Null()
		}
		return ValueOf(v)
	})
	o.props["removeItem"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) > 0 {
			delete(s.items, args[0].String())
		}
		return Undefined()
	})
	o.props["clear"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		_ = args
		s.items = map[string]string{}
		return Undefined()
	})
	o.props["key"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) == 0 {
			return Null()
		}
		i := args[0].Int()
		j := 0
		for k := range s.items {
			if j == i {
				return ValueOf(k)
			}
			j++
		}
		return Null()
	})
	o.props["length"] = ValueOf(len(s.items))
	return o
}

func (w *Window) newDocument() *object {
	doc := &object{props: map[string]Value{}}
	doc.props["body"] = Value{v: w.body.object}
	doc.props["head"] = Value{v: w.head.object}
	doc.props["documentElement"] = Value{v: newNode("html").object}

	doc.props["createElement"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) == 0 {
			return Null()
		}
		n := newNode(strings.ToLower(args[0].String()))
		return Value{v: n.object}
	})

	doc.props["createElementNS"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) < 2 {
			return Null()
		}
		n := newNode(strings.ToLower(args[1].String()))
		return Value{v: n.object}
	})

	doc.props["createTextNode"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		text := ""
		if len(args) > 0 {
			text = args[0].String()
		}
		n := newNode("#text")
		n.nodeValue = text
		n.object.props["nodeValue"] = ValueOf(text)
		return Value{v: n.object}
	})

	doc.props["getElementById"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		if len(args) == 0 {
			return Null()
		}
		return w.GetElementByID(args[0].String())
	})

	doc.props["getElementsByTagName"] = FuncOf(func(this Value, args []Value) Value {
		_ = this
		_ = args
		return Value{v: &object{arr: []Value{}, props: map[string]Value{}}}
	})

	return doc
}

func newNode(tag string) *node {
	n := &node{
		tag:       strings.ToUpper(tag),
		attrs:     map[string]string{},
		children:  []*object{},
		listeners: map[string][]Value{},
	}
	o := &object{props: map[string]Value{}}
	n.object = o
	o.node = n
	o.props["tagName"] = ValueOf(n.tag)
	o.props["children"] = Value{v: &object{arr: []Value{}, props: map[string]Value{}}}
	o.props["getAttribute"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) == 0 {
			return Null()
		}
		key := args[0].String()
		nv := asNode(this)
		if nv == nil {
			return Null()
		}
		v, ok := nv.attrs[key]
		if !ok {
			return Null()
		}
		return ValueOf(v)
	})
	o.props["setAttribute"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) < 2 {
			return Undefined()
		}
		nv := asNode(this)
		if nv == nil {
			return Undefined()
		}
		k := args[0].String()
		v := args[1].String()
		nv.attrs[k] = v
		if k == "id" {
			this.Set("id", ValueOf(v))
		}
		return Undefined()
	})
	o.props["removeAttribute"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) == 0 {
			return Undefined()
		}
		nv := asNode(this)
		if nv == nil {
			return Undefined()
		}
		delete(nv.attrs, args[0].String())
		return Undefined()
	})
	o.props["appendChild"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) == 0 {
			return Undefined()
		}
		p := asNode(this)
		c := asNode(args[0])
		if p == nil || c == nil {
			return Undefined()
		}
		p.appendChild(c)
		return args[0]
	})
	o.props["replaceChild"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) < 2 {
			return Undefined()
		}
		p := asNode(this)
		newChild := asNode(args[0])
		oldChild := asNode(args[1])
		if p == nil || newChild == nil || oldChild == nil {
			return Undefined()
		}
		p.replaceChild(newChild, oldChild)
		return args[0]
	})
	o.props["removeChild"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) == 0 {
			return Undefined()
		}
		p := asNode(this)
		c := asNode(args[0])
		if p == nil || c == nil {
			return Undefined()
		}
		p.removeChild(c)
		return args[0]
	})
	o.props["addEventListener"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) < 2 {
			return Undefined()
		}
		nv := asNode(this)
		if nv != nil {
			ev := args[0].String()
			nv.listeners[ev] = append(nv.listeners[ev], args[1])
		}
		return Undefined()
	})
	o.props["removeEventListener"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) < 2 {
			return Undefined()
		}
		nv := asNode(this)
		if nv == nil {
			return Undefined()
		}
		ev := args[0].String()
		l := nv.listeners[ev]
		out := make([]Value, 0, len(l))
		for _, item := range l {
			if !item.Equal(args[1]) {
				out = append(out, item)
			}
		}
		nv.listeners[ev] = out
		return Undefined()
	})
	o.props["scrollIntoView"] = FuncOf(func(Value, []Value) Value { return Undefined() })
	o.props["dispatchEvent"] = FuncOf(func(this Value, args []Value) Value {
		if len(args) == 0 {
			return ValueOf(false)
		}
		nv := asNode(this)
		if nv == nil {
			return ValueOf(false)
		}
		event := args[0]
		typ := event.Get("type").String()
		for _, handler := range nv.listeners[typ] {
			handler.Invoke(event)
		}
		return ValueOf(true)
	})
	o.props["remove"] = FuncOf(func(this Value, args []Value) Value {
		_ = args
		nv := asNode(this)
		if nv != nil && nv.parent != nil {
			parent := asNode(Value{v: nv.parent})
			if parent != nil {
				parent.removeChild(nv)
			}
		}
		return Undefined()
	})
	return n
}

func (n *node) appendChild(c *node) {
	c.parent = n.object
	n.children = append(n.children, c.object)
	n.syncChildren()
}

func (n *node) replaceChild(newChild, oldChild *node) {
	for i, child := range n.children {
		if child == oldChild.object {
			newChild.parent = n.object
			oldChild.parent = nil
			n.children[i] = newChild.object
			break
		}
	}
	n.syncChildren()
}

func (n *node) removeChild(c *node) {
	for i, child := range n.children {
		if child == c.object {
			c.parent = nil
			n.children = append(n.children[:i], n.children[i+1:]...)
			break
		}
	}
	n.syncChildren()
}

func (n *node) syncChildren() {
	arr := make([]Value, len(n.children))
	for i, child := range n.children {
		arr[i] = Value{v: child}
	}
	n.object.props["children"] = Value{v: &object{arr: arr, props: map[string]Value{}}}
	if len(n.children) > 0 {
		n.object.props["firstChild"] = Value{v: n.children[0]}
		n.object.props["firstElementChild"] = Value{v: n.children[0]}
		n.object.props["lastChild"] = Value{v: n.children[len(n.children)-1]}
	} else {
		n.object.props["firstChild"] = Null()
		n.object.props["firstElementChild"] = Null()
		n.object.props["lastChild"] = Null()
	}
}

func asNode(v Value) *node {
	o, ok := v.v.(*object)
	if !ok {
		return nil
	}
	return o.node
}

func (v Value) Bool() bool {
	if b, ok := v.v.(bool); ok {
		return b
	}
	return false
}

func (v Value) Call(m string, args ...any) Value {
	method := v.Get(m)
	o, ok := method.v.(*object)
	if !ok || o.fn == nil {
		return Undefined()
	}
	converted := make([]Value, len(args))
	for i, arg := range args {
		converted[i] = ValueOf(arg)
	}
	return o.fn(v, converted)
}

func (v Value) Delete(p string) {
	if o, ok := v.v.(*object); ok {
		delete(o.props, p)
	}
}

func (v Value) Equal(w Value) bool {
	switch x := v.v.(type) {
	case *object:
		y, ok := w.v.(*object)
		return ok && x == y
	case string:
		y, ok := w.v.(string)
		return ok && x == y
	case bool:
		y, ok := w.v.(bool)
		return ok && x == y
	case float64:
		y, ok := w.v.(float64)
		return ok && x == y
	default:
		return fmt.Sprintf("%T:%v", v.v, v.v) == fmt.Sprintf("%T:%v", w.v, w.v)
	}
}

func (v Value) Float() float64 {
	switch x := v.v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}

func (v Value) Get(p string) Value {
	if o, ok := v.v.(*object); ok {
		switch p {
		case "length":
			if o.arr != nil {
				return ValueOf(len(o.arr))
			}
			if val, ok := o.props[p]; ok {
				return val
			}
			return ValueOf(0)
		}
		if val, ok := o.props[p]; ok {
			return val
		}
		return Undefined()
	}

	return Undefined()
}

func (v Value) Index(i int) Value {
	if o, ok := v.v.(*object); ok && o.arr != nil {
		if i >= 0 && i < len(o.arr) {
			return o.arr[i]
		}
	}
	return Undefined()
}

func (v Value) InstanceOf(t Value) bool {
	o, ok := t.v.(*object)
	if !ok || o.ctor == nil {
		return false
	}
	return v.Type() == TypeObject
}

func (v Value) Int() int {
	return int(v.Float())
}

func (v Value) Invoke(args ...any) Value {
	o, ok := v.v.(*object)
	if !ok || o.fn == nil {
		return Undefined()
	}
	converted := make([]Value, len(args))
	for i, arg := range args {
		converted[i] = ValueOf(arg)
	}
	return o.fn(Undefined(), converted)
}

func (v Value) IsNaN() bool {
	f, ok := v.v.(float64)
	return ok && math.IsNaN(f)
}

func (v Value) IsNull() bool {
	return v.v == nullSentinel
}

func (v Value) IsUndefined() bool {
	return v.v == undefinedSentinel
}

func (v Value) Length() int {
	if o, ok := v.v.(*object); ok && o.arr != nil {
		return len(o.arr)
	}
	if s, ok := v.v.(string); ok {
		return len(s)
	}
	return 0
}

func (v Value) New(args ...any) Value {
	o, ok := v.v.(*object)
	if !ok {
		return Undefined()
	}
	converted := make([]Value, len(args))
	for i, arg := range args {
		converted[i] = ValueOf(arg)
	}
	if o.ctor != nil {
		return o.ctor(Undefined(), converted)
	}
	if o.fn != nil {
		return o.fn(Undefined(), converted)
	}
	return Value{v: &object{props: map[string]Value{}}}
}

func (v Value) Set(p string, x any) {
	o, ok := v.v.(*object)
	if !ok {
		return
	}
	val := ValueOf(x)
	o.props[p] = val
}

func (v Value) SetIndex(i int, x any) {
	o, ok := v.v.(*object)
	if !ok {
		return
	}
	for len(o.arr) <= i {
		o.arr = append(o.arr, Undefined())
	}
	o.arr[i] = ValueOf(x)
}

func (v Value) String() string {
	switch x := v.v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case *object:
		if text, ok := x.props["innerText"]; ok {
			return text.String()
		}
		if nodeValue, ok := x.props["nodeValue"]; ok {
			return nodeValue.String()
		}
		return "[object Object]"
	default:
		if x == nullSentinel {
			return "null"
		}
		if x == undefinedSentinel {
			return "undefined"
		}
		return fmt.Sprint(x)
	}
}

func (v Value) Truthy() bool {
	switch x := v.v.(type) {
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0 && !math.IsNaN(x)
	case *object:
		return x != nil
	default:
		return x != nil && x != nullSentinel && x != undefinedSentinel
	}
}

func (v Value) Type() Type {
	switch v.v.(type) {
	case nil:
		return TypeUndefined
	case bool:
		return TypeBoolean
	case float64:
		return TypeNumber
	case string:
		return TypeString
	case *object:
		o := v.v.(*object)
		if o.fn != nil || o.ctor != nil {
			return TypeFunction
		}
		return TypeObject
	default:
		if v.v == nullSentinel {
			return TypeNull
		}
		if v.v == undefinedSentinel {
			return TypeUndefined
		}
		return TypeObject
	}
}

func (v Value) Then(f func(Value)) {
	if o, ok := v.v.(*object); ok && o.promise != nil {
		if o.promise.resolved {
			f(o.promise.value)
			return
		}
		o.promise.then = append(o.promise.then, FuncOf(func(this Value, args []Value) Value {
			_ = this
			if len(args) == 0 {
				f(Undefined())
			} else {
				f(args[0])
			}
			return Undefined()
		}))
		return
	}
	f(Undefined())
}

func CopyBytesToGo(dst []byte, src Value) int {
	if src.Type() != TypeObject {
		return 0
	}
	o, ok := src.v.(*object)
	if !ok || o.arr == nil {
		return 0
	}
	n := len(dst)
	if len(o.arr) < n {
		n = len(o.arr)
	}
	for i := 0; i < n; i++ {
		dst[i] = byte(o.arr[i].Int())
	}
	return n
}

func CopyBytesToJS(dst Value, src []byte) int {
	o, ok := dst.v.(*object)
	if !ok {
		return 0
	}
	if len(o.arr) < len(src) {
		o.arr = make([]Value, len(src))
	}
	n := len(src)
	if len(o.arr) < n {
		n = len(o.arr)
	}
	for i := 0; i < n; i++ {
		o.arr[i] = ValueOf(int(src[i]))
	}
	return n
}
