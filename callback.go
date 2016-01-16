package gorm

import (
	"fmt"
)

// defaultCallback hold default callbacks defined by gorm
var defaultCallback = &Callback{}

// Callback contains callbacks that used when CURD objects
//   Field `creates` hold callbacks will be call when creating object
//   Field `updates` hold callbacks will be call when updating object
//   Field `deletes` hold callbacks will be call when deleting object
//   Field `queries` hold callbacks will be call when querying object with query methods like Find, First, Related, Association...
//   Field `rowQueries` hold callbacks will be call when querying object with Row, Rows...
//   Field `processors` hold all callback processors, will be used to generate above callbacks in order
type Callback struct {
	creates    []*func(scope *Scope)
	updates    []*func(scope *Scope)
	deletes    []*func(scope *Scope)
	queries    []*func(scope *Scope)
	rowQueries []*func(scope *Scope)
	processors []*CallbackProcessor
}

// callbackProcessor contains all informations for a callback
type CallbackProcessor struct {
	name      string              // current callback's name
	before    string              // register current callback before a callback
	after     string              // register current callback after a callback
	replace   bool                // replace callbacks with same name
	remove    bool                // delete callbacks with same name
	kind      string              // callback type: create, update, delete, query, row_query
	processor *func(scope *Scope) // callback handler
	parent    *Callback
}

func (c *Callback) addProcessor(kind string) *CallbackProcessor {
	cp := &CallbackProcessor{kind: kind, parent: c}
	c.processors = append(c.processors, cp)
	return cp
}

func (c *Callback) clone() *Callback {
	return &Callback{
		creates:    c.creates,
		updates:    c.updates,
		deletes:    c.deletes,
		queries:    c.queries,
		processors: c.processors,
	}
}

// Create could be used to register callbacks for creating object
//     db.Callback().Create().After("gorm:create").Register("plugin:run_after_create", func(*Scope) {
//       // business logic
//       ...
//
//       // set error if some thing wrong happened, will rollback the creating
//       scope.Err(errors.New("error"))
//     })
func (c *Callback) Create() *CallbackProcessor {
	return c.addProcessor("create")
}

// Update could be used to register callbacks for updating object, refer `Create` for usage
func (c *Callback) Update() *CallbackProcessor {
	return c.addProcessor("update")
}

// Delete could be used to register callbacks for deleting object, refer `Create` for usage
func (c *Callback) Delete() *CallbackProcessor {
	return c.addProcessor("delete")
}

// Query could be used to register callbacks for querying objects with query methods like `Find`, `First`, `Related`, `Association`...
// refer `Create` for usage
func (c *Callback) Query() *CallbackProcessor {
	return c.addProcessor("query")
}

// Query could be used to register callbacks for querying objects with `Row`, `Rows`, refer `Create` for usage
func (c *Callback) RowQuery() *CallbackProcessor {
	return c.addProcessor("row_query")
}

// After insert a new callback after callback `callbackName`, refer `Callbacks.Create`
func (cp *CallbackProcessor) After(callbackName string) *CallbackProcessor {
	cp.after = callbackName
	return cp
}

// Before insert a new callback before callback `callbackName`, refer `Callbacks.Create`
func (cp *CallbackProcessor) Before(callbackName string) *CallbackProcessor {
	cp.before = callbackName
	return cp
}

// Register a new callback, refer `Callbacks.Create`
func (cp *CallbackProcessor) Register(callbackName string, callback func(scope *Scope)) {
	cp.name = callbackName
	cp.processor = &callback
	cp.parent.reorder()
}

// Remove a registered callback
//     db.Callback().Create().Remove("gorm:update_time_stamp_when_create")
func (cp *CallbackProcessor) Remove(callbackName string) {
	fmt.Printf("[info] removing callback `%v` from %v\n", callbackName, fileWithLineNum())
	cp.name = callbackName
	cp.remove = true
	cp.parent.reorder()
}

// Replace a registered callback with new callback
//     db.Callback().Create().Replace("gorm:update_time_stamp_when_create", func(*Scope) {
//		   scope.SetColumn("Created", now)
//		   scope.SetColumn("Updated", now)
//     })
func (cp *CallbackProcessor) Replace(callbackName string, callback func(scope *Scope)) {
	fmt.Printf("[info] replacing callback `%v` from %v\n", callbackName, fileWithLineNum())
	cp.name = callbackName
	cp.processor = &callback
	cp.replace = true
	cp.parent.reorder()
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

// sortProcessors sort callback processors based on its before, after, remove, replace
func sortProcessors(cps []*CallbackProcessor) []*func(scope *Scope) {
	var sortCallbackProcessor func(c *CallbackProcessor)
	var names, sortedNames = []string{}, []string{}

	for _, cp := range cps {
		if index := getRIndex(names, cp.name); index > -1 {
			if !cp.replace && !cp.remove {
				fmt.Printf("[warning] duplicated callback `%v` from %v\n", cp.name, fileWithLineNum())
			}
		}
		names = append(names, cp.name)
	}

	sortCallbackProcessor = func(c *CallbackProcessor) {
		if getRIndex(sortedNames, c.name) > -1 {
			return
		}

		if len(c.before) > 0 {
			if index := getRIndex(sortedNames, c.before); index > -1 {
				sortedNames = append(sortedNames[:index], append([]string{c.name}, sortedNames[index:]...)...)
			} else if index := getRIndex(names, c.before); index > -1 {
				sortedNames = append(sortedNames, c.name)
				sortCallbackProcessor(cps[index])
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
				sortCallbackProcessor(cp)
			} else {
				sortedNames = append(sortedNames, c.name)
			}
		}

		if getRIndex(sortedNames, c.name) == -1 {
			sortedNames = append(sortedNames, c.name)
		}
	}

	for _, cp := range cps {
		sortCallbackProcessor(cp)
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

// reorder all registered processors, and reset CURD callbacks
func (c *Callback) reorder() {
	var creates, updates, deletes, queries, rowQueries []*CallbackProcessor

	for _, processor := range c.processors {
		switch processor.kind {
		case "create":
			creates = append(creates, processor)
		case "update":
			updates = append(updates, processor)
		case "delete":
			deletes = append(deletes, processor)
		case "query":
			queries = append(queries, processor)
		case "row_query":
			rowQueries = append(rowQueries, processor)
		}
	}

	c.creates = sortProcessors(creates)
	c.updates = sortProcessors(updates)
	c.deletes = sortProcessors(deletes)
	c.queries = sortProcessors(queries)
	c.rowQueries = sortProcessors(rowQueries)
}
