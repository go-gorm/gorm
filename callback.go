package gorm

type callback struct {
	creates    []*func()
	updates    []*func()
	deletes    []*func()
	queries    []*func()
	processors []*callback_processor
}

type callback_processor struct {
	name      string
	before    string
	after     string
	replace   bool
	typ       string
	processor *func()
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
	cp.processor = &fc
	cp.callback.sort()
}

func (cp *callback_processor) Remove(name string) {
	cp.Replace(name, func() {})
}

func (cp *callback_processor) Replace(name string, fc func()) {
	cp.name = name
	cp.processor = &fc
	cp.replace = true
	cp.callback.sort()
}

func getIndex(strs []string, str string) int {
	for index, value := range strs {
		if str == value {
			return index
		}
	}
	return -1
}

func sortProcessors(cps []*callback_processor) []*func() {
	var sortCallbackProcessor func(c *callback_processor, force bool)
	var names, sortedNames = []string{}, []string{}

	for _, cp := range cps {
		names = append(names, cp.name)
	}

	sortCallbackProcessor = func(c *callback_processor, force bool) {
		if getIndex(sortedNames, c.name) > -1 {
			return
		}

		if len(c.before) > 0 {
			if index := getIndex(sortedNames, c.before); index > -1 {
				sortedNames = append(sortedNames[:index], append([]string{c.name}, sortedNames[index:]...)...)
			} else if index := getIndex(names, c.before); index > -1 {
				sortedNames = append(sortedNames, c.name)
				sortCallbackProcessor(cps[index], true)
			} else {
				sortedNames = append(sortedNames, c.name)
			}
		}

		if len(c.after) > 0 {
			if index := getIndex(sortedNames, c.after); index > -1 {
				sortedNames = append(sortedNames[:index+1], append([]string{c.name}, sortedNames[index+1:]...)...)
			} else if index := getIndex(names, c.after); index > -1 {
				cp := cps[index]
				if len(cp.before) == 0 {
					cp.before = c.name
				}
				sortCallbackProcessor(cp, true)
			} else {
				sortedNames = append(sortedNames, c.name)
			}
		}

		if getIndex(sortedNames, c.name) == -1 && force {
			sortedNames = append(sortedNames, c.name)
		}
	}

	for _, cp := range cps {
		sortCallbackProcessor(cp, false)
	}

	var funcs = []*func(){}
	var sortedFuncs = []*func(){}
	for _, name := range sortedNames {
		index := getIndex(names, name)
		sortedFuncs = append(sortedFuncs, cps[index].processor)
	}

	for _, cp := range cps {
		if sindex := getIndex(sortedNames, cp.name); sindex == -1 {
			funcs = append(funcs, cp.processor)
		}
	}

	return append(sortedFuncs, funcs...)
}

func (c *callback) sort() {
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

	c.creates = sortProcessors(creates)
	c.updates = sortProcessors(updates)
	c.deletes = sortProcessors(deletes)
	c.queries = sortProcessors(queries)
}

var DefaultCallback = &callback{processors: []*callback_processor{}}
