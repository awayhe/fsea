package gwf

import (
	"net/http"
	"strings"
)

// 使用一个Context对象把所有上下文都包括掉
type Context struct {
	w http.ResponseWriter
	r *http.Request

	path []string
}

type Module interface {
	Action(*Context)
}

type Dispatcher struct {
	// module name => module interface
	modules map[string]Module
}

func (c *Context) Request() *http.Request {
	return c.r
}

func (c *Context) Writer() http.ResponseWriter {
	return c.w
}

func (c *Context) Init(w http.ResponseWriter, r *http.Request) {
	c.w, c.r = w, r

	path := strings.Split(r.URL.Path, "/")
	c.path = path[1:]
}

func (c *Context) Path(index int) (string, bool) {
	if index < 0 || index >= len(c.path) {
		return "", false
	} else {
		return c.path[index], true
	}
}

func (c *Context) Depth() int {
	return len(c.path)
}

// 增加对"/name"的处理器
func (d *Dispatcher) AddModule(name string, m Module) {
	if nil == d.modules {
		d.modules = make(map[string]Module)
	}
	d.modules[name] = m
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var c Context
	c.Init(w, r)

	if name, ok := c.Path(0); ok {
		if module, ok := d.modules[name]; ok && module != nil {
			(module).Action(&c)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}
