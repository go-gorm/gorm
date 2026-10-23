// Package logger defines GORM's logging interface and built-in implementations.
// Configure a logger when opening a DB through gorm.Config.Logger, or select a
// logger for a session through gorm.Session.Logger. LogMode returns a configured
// copy in the built-in implementations; shared output writers remain shared.
//
// # Configuration
//
// New accepts a printf-compatible Writer, such as a standard log.Logger, and a
// Config. Configuration is used as supplied: New does not inherit Default's Warn
// level, 200ms slow threshold, or color setting. A typical setup is:
//
//	output := log.New(os.Stdout, "", log.LstdFlags)
//	sqlLogger := logger.New(output, logger.Config{
//	    LogLevel:             logger.Warn,
//	    SlowThreshold:        200 * time.Millisecond,
//	    ParameterizedQueries: true,
//	})
//	db, err := gorm.Open(dialector, &gorm.Config{Logger: sqlLogger})
//
// Handle err before using db. The application owns output's underlying files or
// other resources; GORM does not close them. Supply a concurrency-safe writer or
// handler when a DB is used by multiple goroutines. Prefer per-DB or per-session
// configuration over changing package variables while queries are running.
//
// NewSlogLogger adapts log/slog on Go 1.21 and later. Its handler controls final
// output and level enablement. Its Trace filtering differs from the printf
// implementation for error and slow-query branches; see the adapter's methods.
//
// # Traces and sensitive values
//
// Trace receives the operation context, start time, error, and a callback that
// formats SQL and returns RowsAffected. A row count of -1 means unavailable.
// Custom implementations can defer calling the callback until they need SQL.
// The printf implementation prioritizes failed queries, then slow queries, then
// ordinary Info traces. SlowThreshold classifies elapsed time; it does not impose
// a query timeout. Ignoring ErrRecordNotFound changes logging, not DB.Error.
//
// ParameterizedQueries suppresses bound values in GORM's trace formatting. It does
// not remove literals already embedded in SQL, nor redact arbitrary diagnostic
// messages. ExplainSQL produces a readable approximation for logs; its output
// must not be executed as a substitute for driver parameter binding. Recorder
// captures mutable state for one operation; create a fresh recorder with New for
// independent captures.
//
// See https://gorm.io/docs/logger.html for configuration and custom loggers.
package logger
