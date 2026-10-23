package gorm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// initializeCallbacks creates empty processors for the six operation families. The
// dialector later registers their default callbacks and clause order.
func initializeCallbacks(db *DB) *callbacks {
	return &callbacks{
		processors: map[string]*processor{
			"create": {db: db},
			"query":  {db: db},
			"update": {db: db},
			"delete": {db: db},
			"row":    {db: db},
			"raw":    {db: db},
		},
	}
}

// callbacks owns the shared operation registry returned by DB.Callback. Session copies
// share it; registration must finish before concurrent execution.
type callbacks struct {
	// processors maps operation family names to their compiled callback pipelines.
	processors map[string]*processor
}

// processor stores callback registrations, their compiled execution order, and the
// default SQL clause order for one operation family. Mutating registrations is not
// synchronized with execution.
type processor struct {
	// db is the initialization handle used to evaluate registration predicates and log
	// compile errors.
	db *DB
	// Clauses supplies SQL clause order when the executing statement has no explicit
	// BuildClauses.
	Clauses []string
	// fns is the compiled handler sequence invoked by Execute, including handlers
	// responsible for checking db.Error.
	fns []func(*DB)
	// callbacks retains registrations after match filtering and removals; compile derives
	// fns from this list.
	callbacks []*callback
}

// callback is a registration builder carrying a handler, ordering constraints, and
// optional match/remove/replace instructions. Register, Remove, or Replace appends it
// and recompiles the processor.
type callback struct {
	// name identifies the handler within its operation family; use a plugin-specific prefix
	// to avoid collisions.
	name string
	// before requests ordering before a named callback; "*" requests the beginning of the
	// pipeline.
	before string
	// after requests ordering after a named callback; "*" requests the end of the pipeline.
	after string
	// remove marks this registration as a request to remove handlers with name during
	// compilation.
	remove bool
	// replace indicates an intentional replacement of the named handler and suppresses
	// duplicate-registration warnings.
	replace bool
	// match is evaluated against the processor's initialization DB during compilation, not
	// for each query.
	match func(*DB) bool
	// handler is invoked with the executing DB and should record failures through AddError.
	handler func(*DB)
	// processor receives this registration and compiles the resulting order.
	processor *processor
}

// Create returns the shared callback processor for inserts, including create hooks and
// association saving. Register plugins before concurrent use; changes affect all
// sessions sharing this registry.
func (cs *callbacks) Create() *processor {
	return cs.processors["create"]
}

// Query returns the shared callback processor for model queries, including scanning,
// preloading, and AfterFind. Register plugins before concurrent use; changes affect all
// sessions sharing this registry.
func (cs *callbacks) Query() *processor {
	return cs.processors["query"]
}

// Update returns the shared callback processor for updates, including update hooks and
// timestamps. Register plugins before concurrent use; changes affect all sessions
// sharing this registry.
func (cs *callbacks) Update() *processor {
	return cs.processors["update"]
}

// Delete returns the shared callback processor for deletes, including soft deletion and
// selected associations. Register plugins before concurrent use; changes affect all
// sessions sharing this registry.
func (cs *callbacks) Delete() *processor {
	return cs.processors["delete"]
}

// Row returns the shared callback processor for manual Row/Rows queries without normal
// model scanning or preloading. Register plugins before concurrent use; changes affect
// all sessions sharing this registry.
func (cs *callbacks) Row() *processor {
	return cs.processors["row"]
}

// Raw returns the shared callback processor for raw Exec statements without model write
// hooks. Register plugins before concurrent use; changes affect all sessions sharing
// this registry.
func (cs *callbacks) Raw() *processor {
	return cs.processors["raw"]
}

// Execute consumes queued scopes, applies a destination StatementModifier, establishes
// model/schema/reflection state and clause order, then runs all compiled handlers.
// Handlers must check Error themselves; the loop does not stop on it. The operation is
// traced and SQL/Vars are cleared afterward unless DryRun. It returns the resulting
// mutable DB handle.
func (p *processor) Execute(db *DB) *DB {
	// call scopes
	for len(db.Statement.scopes) > 0 {
		db = db.executeScopes()
	}

	var (
		curTime           = time.Now()
		stmt              = db.Statement
		resetBuildClauses bool
	)

	if len(stmt.BuildClauses) == 0 {
		stmt.BuildClauses = p.Clauses
		resetBuildClauses = true
	}

	if optimizer, ok := stmt.Dest.(StatementModifier); ok {
		optimizer.ModifyStatement(stmt)
	}

	if db.DefaultContextTimeout > 0 {
		if _, ok := stmt.Context.Deadline(); !ok {
			stmt.Context, _ = context.WithTimeout(stmt.Context, db.DefaultContextTimeout)
		}
	}

	// assign model values
	if stmt.Model == nil {
		stmt.Model = stmt.Dest
	} else if stmt.Dest == nil {
		stmt.Dest = stmt.Model
	}

	// parse model values
	if stmt.Model != nil {
		if err := stmt.Parse(stmt.Model); err != nil && (!errors.Is(err, schema.ErrUnsupportedDataType) || (stmt.Table == "" && stmt.TableExpr == nil && stmt.SQL.Len() == 0)) {
			if errors.Is(err, schema.ErrUnsupportedDataType) && stmt.Table == "" && stmt.TableExpr == nil {
				db.AddError(fmt.Errorf("%w: Table not set, please set it like: db.Model(&user) or db.Table(\"users\")", err))
			} else {
				db.AddError(err)
			}
		}
	}

	// assign stmt.ReflectValue
	if stmt.Dest != nil {
		stmt.ReflectValue = reflect.ValueOf(stmt.Dest)
		for stmt.ReflectValue.Kind() == reflect.Ptr {
			if stmt.ReflectValue.IsNil() && stmt.ReflectValue.CanAddr() {
				stmt.ReflectValue.Set(reflect.New(stmt.ReflectValue.Type().Elem()))
			}

			stmt.ReflectValue = stmt.ReflectValue.Elem()
		}
		if !stmt.ReflectValue.IsValid() {
			db.AddError(ErrInvalidValue)
		}
	}

	for _, f := range p.fns {
		f(db)
	}

	if stmt.SQL.Len() > 0 {
		db.Logger.Trace(stmt.Context, curTime, func() (string, int64) {
			sql, vars := stmt.SQL.String(), stmt.Vars
			if filter, ok := db.Logger.(ParamsFilter); ok {
				sql, vars = filter.ParamsFilter(stmt.Context, stmt.SQL.String(), stmt.Vars...)
			}
			return db.Dialector.Explain(sql, vars...), db.RowsAffected
		}, db.Error)
	}

	if !stmt.DB.DryRun {
		stmt.SQL.Reset()
		stmt.Vars = nil
	}

	if resetBuildClauses {
		stmt.BuildClauses = nil
	}

	return db
}

// Get returns the last registered, non-removed handler with name, or nil when absent.
// Configure the registry before concurrent lookups and execution.
func (p *processor) Get(name string) func(*DB) {
	for i := len(p.callbacks) - 1; i >= 0; i-- {
		if v := p.callbacks[i]; v.name == name && !v.remove {
			return v.handler
		}
	}
	return nil
}

// Before requests execution before the named handler. Use "*" for the beginning of the
// pipeline. It returns a registration builder; Register applies it and reports ordering
// conflicts.
func (p *processor) Before(name string) *callback {
	return &callback{before: name, processor: p}
}

// After requests execution after the named handler. Use "*" for the end of the
// pipeline. It returns a registration builder; Register applies it and reports ordering
// conflicts.
func (p *processor) After(name string) *callback {
	return &callback{after: name, processor: p}
}

// Match creates a registration whose predicate is evaluated against the initialization
// DB when the processor compiles. It is not a per-operation filter; put
// request-dependent conditions in the handler.
func (p *processor) Match(fc func(*DB) bool) *callback {
	return &callback{match: fc, processor: p}
}

// Register adds a named handler and recompiles execution order. Use a unique
// plugin-prefixed name and check the returned error. The registry is shared across
// sessions and must not be changed concurrently with use.
func (p *processor) Register(name string, fn func(*DB)) error {
	return (&callback{processor: p}).Register(name, fn)
}

// Remove marks a named handler for removal and recompiles the processor. This affects
// all sessions sharing the registry; removing a default handler can disable core ORM
// behavior.
func (p *processor) Remove(name string) error {
	return (&callback{processor: p}).Remove(name)
}

// Replace installs a new handler for an existing name and recompiles order. The new
// handler must fulfill any core responsibilities of the old one. Check compilation
// errors and configure before concurrent use.
func (p *processor) Replace(name string, fn func(*DB)) error {
	return (&callback{processor: p}).Replace(name, fn)
}

// compile filters registrations using their match predicates, removes marked names, and
// computes the execution order. Ordering conflicts are returned and logged. Compilation
// mutates the registry and is not transactional if an error occurs.
func (p *processor) compile() (err error) {
	var callbacks []*callback
	removedMap := map[string]bool{}
	for _, callback := range p.callbacks {
		if callback.match == nil || callback.match(p.db) {
			callbacks = append(callbacks, callback)
		}
		if callback.remove {
			removedMap[callback.name] = true
		}
	}

	if len(removedMap) > 0 {
		callbacks = removeCallbacks(callbacks, removedMap)
	}
	p.callbacks = callbacks

	if p.fns, err = sortCallbacks(p.callbacks); err != nil {
		p.db.Logger.Error(context.Background(), "Got error when compile callbacks, got %v", err)
	}
	return
}

// Before requests execution before the named handler. Use "*" for the beginning of the
// pipeline. It returns a registration builder; Register applies it and reports ordering
// conflicts.
func (c *callback) Before(name string) *callback {
	c.before = name
	return c
}

// After requests execution after the named handler. Use "*" for the end of the
// pipeline. It returns a registration builder; Register applies it and reports ordering
// conflicts.
func (c *callback) After(name string) *callback {
	c.after = name
	return c
}

// Register adds a named handler and recompiles execution order. Use a unique
// plugin-prefixed name and check the returned error. The registry is shared across
// sessions and must not be changed concurrently with use.
func (c *callback) Register(name string, fn func(*DB)) error {
	c.name = name
	c.handler = fn
	c.processor.callbacks = append(c.processor.callbacks, c)
	return c.processor.compile()
}

// Remove marks a named handler for removal and recompiles the processor. This affects
// all sessions sharing the registry; removing a default handler can disable core ORM
// behavior.
func (c *callback) Remove(name string) error {
	c.processor.db.Logger.Warn(context.Background(), "removing callback `%s` from %s\n", name, utils.FileWithLineNum())
	c.name = name
	c.remove = true
	c.processor.callbacks = append(c.processor.callbacks, c)
	return c.processor.compile()
}

// Replace installs a new handler for an existing name and recompiles order. The new
// handler must fulfill any core responsibilities of the old one. Check compilation
// errors and configure before concurrent use.
func (c *callback) Replace(name string, fn func(*DB)) error {
	c.processor.db.Logger.Info(context.Background(), "replacing callback `%s` from %s\n", name, utils.FileWithLineNum())
	c.name = name
	c.handler = fn
	c.replace = true
	c.processor.callbacks = append(c.processor.callbacks, c)
	return c.processor.compile()
}

// getRIndex returns the last index of str in strs, or -1 if absent. Callback ordering
// uses the most recent registration for a name.
func getRIndex(strs []string, str string) int {
	for i := len(strs) - 1; i >= 0; i-- {
		if strs[i] == str {
			return i
		}
	}
	return -1
}

// sortCallbacks resolves before/after constraints, including "*", into handler order
// and reports conflicts. Duplicate names select the last registered handler; unmarked
// duplicates produce a warning. Resolution can update ordering hints in the supplied
// registrations.
func sortCallbacks(cs []*callback) (fns []func(*DB), err error) {
	var (
		names, sorted []string
		sortCallback  func(*callback) error
	)
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[j].before == "*" && cs[i].before != "*" {
			return true
		}
		if cs[j].after == "*" && cs[i].after != "*" {
			return true
		}
		return false
	})

	for _, c := range cs {
		// show warning message the callback name already exists
		if idx := getRIndex(names, c.name); idx > -1 && !c.replace && !c.remove && !cs[idx].remove {
			c.processor.db.Logger.Warn(context.Background(), "duplicated callback `%s` from %s\n", c.name, utils.FileWithLineNum())
		}
		names = append(names, c.name)
	}

	sortCallback = func(c *callback) error {
		if c.before != "" { // if defined before callback
			if c.before == "*" && len(sorted) > 0 {
				if curIdx := getRIndex(sorted, c.name); curIdx == -1 {
					sorted = append([]string{c.name}, sorted...)
				}
			} else if sortedIdx := getRIndex(sorted, c.before); sortedIdx != -1 {
				if curIdx := getRIndex(sorted, c.name); curIdx == -1 {
					// if before callback already sorted, append current callback just after it
					sorted = append(sorted[:sortedIdx], append([]string{c.name}, sorted[sortedIdx:]...)...)
				} else if curIdx > sortedIdx {
					return fmt.Errorf("conflicting callback %s with before %s", c.name, c.before)
				}
			} else if idx := getRIndex(names, c.before); idx != -1 {
				// if before callback exists
				cs[idx].after = c.name
			}
		}

		if c.after != "" { // if defined after callback
			if c.after == "*" && len(sorted) > 0 {
				if curIdx := getRIndex(sorted, c.name); curIdx == -1 {
					sorted = append(sorted, c.name)
				}
			} else if sortedIdx := getRIndex(sorted, c.after); sortedIdx != -1 {
				if curIdx := getRIndex(sorted, c.name); curIdx == -1 {
					// if after callback sorted, append current callback to last
					sorted = append(sorted, c.name)
				} else if curIdx < sortedIdx {
					return fmt.Errorf("conflicting callback %s with after %s", c.name, c.after)
				}
			} else if idx := getRIndex(names, c.after); idx != -1 {
				// if after callback exists but haven't sorted
				// set after callback's before callback to current callback
				after := cs[idx]

				if after.before == "" {
					after.before = c.name
				}

				if err := sortCallback(after); err != nil {
					return err
				}

				if err := sortCallback(c); err != nil {
					return err
				}
			}
		}

		// if current callback haven't been sorted, append it to last
		if getRIndex(sorted, c.name) == -1 {
			sorted = append(sorted, c.name)
		}

		return nil
	}

	for _, c := range cs {
		if err = sortCallback(c); err != nil {
			return
		}
	}

	for _, name := range sorted {
		if idx := getRIndex(names, name); !cs[idx].remove {
			fns = append(fns, cs[idx].handler)
		}
	}

	return
}

// removeCallbacks returns registrations whose names are absent from nameMap, preserving
// their existing order. It does not execute handlers.
func removeCallbacks(cs []*callback, nameMap map[string]bool) []*callback {
	callbacks := make([]*callback, 0, len(cs))
	for _, callback := range cs {
		if nameMap[callback.name] {
			continue
		}
		callbacks = append(callbacks, callback)
	}
	return callbacks
}
