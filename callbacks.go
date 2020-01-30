package gorm

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm/logger"
	"github.com/jinzhu/gorm/utils"
)

// Callbacks gorm callbacks manager
type Callbacks struct {
	creates    []func(*DB)
	queries    []func(*DB)
	updates    []func(*DB)
	deletes    []func(*DB)
	row        []func(*DB)
	raw        []func(*DB)
	db         *DB
	processors []*processor
}

type processor struct {
	kind      string
	name      string
	before    string
	after     string
	remove    bool
	replace   bool
	match     func(*DB) bool
	handler   func(*DB)
	callbacks *Callbacks
}

func (cs *Callbacks) Create() *processor {
	return &processor{callbacks: cs, kind: "create"}
}

func (cs *Callbacks) Query() *processor {
	return &processor{callbacks: cs, kind: "query"}
}

func (cs *Callbacks) Update() *processor {
	return &processor{callbacks: cs, kind: "update"}
}

func (cs *Callbacks) Delete() *processor {
	return &processor{callbacks: cs, kind: "delete"}
}

func (cs *Callbacks) Row() *processor {
	return &processor{callbacks: cs, kind: "row"}
}

func (cs *Callbacks) Raw() *processor {
	return &processor{callbacks: cs, kind: "raw"}
}

func (p *processor) Before(name string) *processor {
	p.before = name
	return p
}

func (p *processor) After(name string) *processor {
	p.after = name
	return p
}

func (p *processor) Match(fc func(*DB) bool) *processor {
	p.match = fc
	return p
}

func (p *processor) Get(name string) func(*DB) {
	for i := len(p.callbacks.processors) - 1; i >= 0; i-- {
		if v := p.callbacks.processors[i]; v.name == name && v.kind == v.kind && !v.remove {
			return v.handler
		}
	}
	return nil
}

func (p *processor) Register(name string, fn func(*DB)) {
	p.name = name
	p.handler = fn
	p.callbacks.processors = append(p.callbacks.processors, p)
	p.callbacks.compile(p.callbacks.db)
}

func (p *processor) Remove(name string) {
	logger.Default.Info("removing callback `%v` from %v\n", name, utils.FileWithLineNum())
	p.name = name
	p.remove = true
	p.callbacks.processors = append(p.callbacks.processors, p)
	p.callbacks.compile(p.callbacks.db)
}

func (p *processor) Replace(name string, fn func(*DB)) {
	logger.Default.Info("[info] replacing callback `%v` from %v\n", name, utils.FileWithLineNum())
	p.name = name
	p.handler = fn
	p.replace = true
	p.callbacks.processors = append(p.callbacks.processors, p)
	p.callbacks.compile(p.callbacks.db)
}

// getRIndex get right index from string slice
func getRIndex(strs []string, str string) int {
	for i := len(strs) - 1; i >= 0; i-- {
		if strs[i] == str {
			return i
		}
	}
	return -1
}

func sortProcessors(ps []*processor) []func(*DB) {
	var (
		allNames, sortedNames []string
		sortProcessor         func(*processor) error
	)

	for _, p := range ps {
		// show warning message the callback name already exists
		if idx := getRIndex(allNames, p.name); idx > -1 && !p.replace && !p.remove && !ps[idx].remove {
			log.Printf("[warning] duplicated callback `%v` from %v\n", p.name, utils.FileWithLineNum())
		}
		allNames = append(allNames, p.name)
	}

	sortProcessor = func(p *processor) error {
		if getRIndex(sortedNames, p.name) == -1 { // if not sorted
			if p.before != "" { // if defined before callback
				if sortedIdx := getRIndex(sortedNames, p.before); sortedIdx != -1 {
					if curIdx := getRIndex(sortedNames, p.name); curIdx != -1 || true {
						// if before callback already sorted, append current callback just after it
						sortedNames = append(sortedNames[:sortedIdx], append([]string{p.name}, sortedNames[sortedIdx:]...)...)
					} else if curIdx > sortedIdx {
						return fmt.Errorf("conflicting callback %v with before %v", p.name, p.before)
					}
				} else if idx := getRIndex(allNames, p.before); idx != -1 {
					// if before callback exists
					ps[idx].after = p.name
				}
			}

			if p.after != "" { // if defined after callback
				if sortedIdx := getRIndex(sortedNames, p.after); sortedIdx != -1 {
					// if after callback sorted, append current callback to last
					sortedNames = append(sortedNames, p.name)
				} else if idx := getRIndex(allNames, p.after); idx != -1 {
					// if after callback exists but haven't sorted
					// set after callback's before callback to current callback
					if after := ps[idx]; after.before == "" {
						after.before = p.name
						sortProcessor(after)
					}
				}
			}

			// if current callback haven't been sorted, append it to last
			if getRIndex(sortedNames, p.name) == -1 {
				sortedNames = append(sortedNames, p.name)
			}
		}

		return nil
	}

	for _, p := range ps {
		sortProcessor(p)
	}

	var fns []func(*DB)
	for _, name := range sortedNames {
		if idx := getRIndex(allNames, name); !ps[idx].remove {
			fns = append(fns, ps[idx].handler)
		}
	}

	return fns
}

// compile processors
func (cs *Callbacks) compile(db *DB) {
	processors := map[string][]*processor{}
	for _, p := range cs.processors {
		if p.name != "" {
			if p.match == nil || p.match(db) {
				processors[p.kind] = append(processors[p.kind], p)
			}
		}
	}

	for name, ps := range processors {
		switch name {
		case "create":
			cs.creates = sortProcessors(ps)
		case "query":
			cs.queries = sortProcessors(ps)
		case "update":
			cs.updates = sortProcessors(ps)
		case "delete":
			cs.deletes = sortProcessors(ps)
		case "row":
			cs.row = sortProcessors(ps)
		case "raw":
			cs.raw = sortProcessors(ps)
		}
	}
}
