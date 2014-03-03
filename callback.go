package gorm

import (
	"fmt"
)

type callback struct {
	creates    []*func(scope *Scope)
	updates    []*func(scope *Scope)
	deletes    []*func(scope *Scope)
	queries    []*func(scope *Scope)
	processors []*callback_processor
}

type callback_processor struct {
	name      string
	before    string
	after     string
	replace   bool
	remove    bool
	typ       string
	processor *func(scope *Scope)
	callback  *callback
}

func (c *callback) addProcessor(typ string) *callback_processor {
	cp := &callback_processor{typ: typ, callback: c}
	c.processors = append(c.processors, cp)
	return cp
}

func (c *callback) clone() *callback {
	return &callback{processors: c.processors}
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

func (cp *callback_processor) Register(name string, fc func(scope *Scope)) {
	cp.name = name
	cp.processor = &fc
	cp.callback.sort()
}

func (cp *callback_processor) Remove(name string) {
	cp.name = name
	cp.remove = true
	cp.callback.sort()
}

func (cp *callback_processor) Replace(name string, fc func(scope *Scope)) {
	cp.name = name
	cp.processor = &fc
	cp.replace = true
	cp.callback.sort()
}

func getRIndex(strs []string, str string) int {
	for i := len(strs) - 1; i >= 0; i-- {
		if strs[i] == str {
			return i
		}
	}
	return -1
}

func sortProcessors(cps []*callback_processor) []*func(scope *Scope) {
	var sortCallbackProcessor func(c *callback_processor, force bool)
	var names, sortedNames = []string{}, []string{}

	for _, cp := range cps {
		if index := getRIndex(names, cp.name); index > -1 {
			if cp.replace {
				fmt.Printf("[info] replacing callback `%v` from %v\n", cp.name, fileWithLineNum())
			} else if cp.remove {
				fmt.Printf("[info] removing callback `%v` from %v\n", cp.name, fileWithLineNum())
			} else {
				fmt.Printf("[warning] duplicated callback `%v` from %v\n", cp.name, fileWithLineNum())
			}
		}
		names = append(names, cp.name)
	}

	sortCallbackProcessor = func(c *callback_processor, force bool) {
		if getRIndex(sortedNames, c.name) > -1 {
			return
		}

		if len(c.before) > 0 {
			if index := getRIndex(sortedNames, c.before); index > -1 {
				sortedNames = append(sortedNames[:index], append([]string{c.name}, sortedNames[index:]...)...)
			} else if index := getRIndex(names, c.before); index > -1 {
				sortedNames = append(sortedNames, c.name)
				sortCallbackProcessor(cps[index], true)
			} else {
				sortedNames = append(sortedNames, c.name)
			}
		}

		if len(c.after) > 0 {
			if index := getRIndex(sortedNames, c.after); index > -1 {
				sortedNames = append(sortedNames[:index+1], append([]string{c.name}, sortedNames[index+1:]...)...)
			} else if index := getRIndex(names, c.after); index > -1 {
				cp := cps[index]
				if len(cp.before) == 0 {
					cp.before = c.name
				}
				sortCallbackProcessor(cp, true)
			} else {
				sortedNames = append(sortedNames, c.name)
			}
		}

		if getRIndex(sortedNames, c.name) == -1 && force {
			sortedNames = append(sortedNames, c.name)
		}
	}

	for _, cp := range cps {
		sortCallbackProcessor(cp, false)
	}

	var funcs = []*func(scope *Scope){}
	var sortedFuncs = []*func(scope *Scope){}
	for _, name := range sortedNames {
		index := getRIndex(names, name)
		if !cps[index].remove {
			sortedFuncs = append(sortedFuncs, cps[index].processor)
		}
	}

	for _, cp := range cps {
		if sindex := getRIndex(sortedNames, cp.name); sindex == -1 {
			if !cp.remove {
				funcs = append(funcs, cp.processor)
			}
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
