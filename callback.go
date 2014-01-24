package gorm

type callback struct {
	create     []func()
	update     []func()
	delete     []func()
	query      []func()
	processors []*callback_processor
}

type callback_processor struct {
	name      string
	before    string
	after     string
	replace   bool
	typ       string
	processor func()
	callback  *callback
}

func (c *callback) addProcessor(typ string) *callback_processor {
	cp := &callback_processor{typ: typ, callback: c}
	c.processors = append(c.processors, cp)
	return cp
}

func (c *callback) Create() *callback_processor {
	return c.addProcessor("create")
}

func (c *callback) Update() *callback_processor {
	return c.addProcessor("update")
}

func (c *callback) Delete() *callback_processor {
	return c.addProcessor("delete")
}

func (c *callback) Query() *callback_processor {
	return c.addProcessor("query")
}

func (c *callback) Sort() {
	creates, updates, deletes, queries := []*callback_processor{}, []*callback_processor{}, []*callback_processor{}, []*callback_processor{}

	for _, processor := range c.processors {
		switch processor.typ {
		case "create":
			creates = append(creates, processor)
		case "update":
			updates = append(updates, processor)
		case "delete":
			deletes = append(deletes, processor)
		case "query":
			queries = append(queries, processor)
		}
	}
}

func (cp *callback_processor) Before(name string) *callback_processor {
	cp.before = name
	return cp
}

func (cp *callback_processor) After(name string) *callback_processor {
	cp.after = name
	return cp
}

func (cp *callback_processor) Register(name string, fc func()) {
	cp.name = name
	cp.processor = fc
	cp.callback.Sort()
}

func (cp *callback_processor) Remove(name string) {
	cp.Replace(name, func() {})
}

func (cp *callback_processor) Replace(name string, fc func()) {
	cp.name = name
	cp.processor = fc
	cp.replace = true
	cp.callback.Sort()
}

var DefaultCallback = &callback{processors: []*callback_processor{}}
