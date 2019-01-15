package logp

import (
	"flag"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	golog "log"
	"os"
	"path"
	"sync/atomic"
	"unsafe"
)

var (
	_log unsafe.Pointer // Pointer to a coreLogger. Access via atomic.LoadPointer.
)

func init() {
	storeLogger(&coreLogger{
		selectors:    map[string]struct{}{},
		rootLogger:   zap.NewNop(),
		globalLogger: zap.NewNop(),
		logger:       newLogger(zap.NewNop(), ""),
	})
}

type coreLogger struct {
	selectors    map[string]struct{}    // Set of enabled debug selectors.
	rootLogger   *zap.Logger            // Root logger without any options configured.
	globalLogger *zap.Logger            // Logger used by legacy global functions (e.g. logp.Info).
	logger       *Logger                // Logger that is the basis for all logp.Loggers.
	observedLogs *observer.ObservedLogs // Contains events generated while in observation mode (a testing mode).
}

// Configure configures the logp package.
func Configure(cfg Config) error {
	var (
		sink         zapcore.Core
		observedLogs *observer.ObservedLogs
		err          error
	)

	// Build a single output (stderr has priority if more than one are enabled).
	switch {
	case cfg.toObserver:
		sink, observedLogs = observer.New(cfg.Level.zapLevel())
	case cfg.toIODiscard:
		sink, err = makeDiscardOutput(cfg)
	case cfg.ToStderr:
		sink, err = makeStderrOutput(cfg)
	case cfg.ToSyslog:
		sink, err = makeSyslogOutput(cfg)
	case cfg.ToEventLog:
		sink, err = makeEventLogOutput(cfg)
	case cfg.ToFiles:
		fallthrough
	default:
		sink, err = makeFileOutput(cfg)
	}
	if err != nil {
		return errors.Wrap(err, "failed to build log output")
	}

	// Enabled selectors when debug is enabled.
	selectors := make(map[string]struct{}, len(cfg.Selectors))
	if cfg.Level.Enabled(DebugLevel) && len(cfg.Selectors) > 0 {
		for _, sel := range cfg.Selectors {
			selectors[sel] = struct{}{}
		}

		// Default to all enabled if no selectors are specified.
		if len(selectors) == 0 {
			selectors["*"] = struct{}{}
		}

		if _, enabled := selectors["stdlog"]; !enabled {
			// Disable standard logging by default (this is sometimes used by
			// libraries and we don't want their spam).
			golog.SetOutput(ioutil.Discard)
		}

		sink = selectiveWrapper(sink, selectors)
	}

	root := zap.New(sink, makeOptions(cfg)...)
	storeLogger(&coreLogger{
		selectors:    selectors,
		rootLogger:   root,
		globalLogger: root.WithOptions(zap.AddCallerSkip(1)),
		logger:       newLogger(root, ""),
		observedLogs: observedLogs,
	})
	return nil
}

// DevelopmentSetup configures the logger in development mode at debug level.
// By default the output goes to stderr.
func DevelopmentSetup(options ...Option) error {
	cfg := Config{
		Level:       DebugLevel,
		ToStderr:    true,
		development: true,
		addCaller:   true,
	}
	for _, apply := range options {
		apply(&cfg)
	}
	return Configure(cfg)
}

// TestingSetup configures logging by calling DevelopmentSetup if and only if
// verbose testing is enabled (as in 'go test -v').
func TestingSetup(options ...Option) error {
	// Use the flag to avoid a dependency on the testing package.
	f := flag.Lookup("test.v")
	if f != nil && f.Value.String() == "true" {
		return DevelopmentSetup(options...)
	}
	return nil
}

// ObserverLogs provides the list of logs generated during the observation
// process.
func ObserverLogs() *observer.ObservedLogs {
	return loadLogger().observedLogs
}

// Sync flushes any buffered log entries. Applications should take care to call
// Sync before exiting.
func Sync() error {
	return loadLogger().rootLogger.Sync()
}

func makeOptions(cfg Config) []zap.Option {
	var options []zap.Option
	if cfg.addCaller {
		options = append(options, zap.AddCaller())
	}
	if cfg.development {
		options = append(options, zap.Development())
	}
	return options
}

func makeStderrOutput(cfg Config) (zapcore.Core, error) {
	stderr := zapcore.Lock(os.Stderr)
	return zapcore.NewCore(buildEncoder(cfg), stderr, cfg.Level.zapLevel()), nil
}

func makeDiscardOutput(cfg Config) (zapcore.Core, error) {
	discard := zapcore.AddSync(ioutil.Discard)
	return zapcore.NewCore(buildEncoder(cfg), discard, cfg.Level.zapLevel()), nil
}

func makeSyslogOutput(cfg Config) (zapcore.Core, error) {
	return newSyslog(buildEncoder(cfg), cfg.Level.zapLevel())
}

func makeEventLogOutput(cfg Config) (zapcore.Core, error) {
	return newEventLog(buildEncoder(cfg), cfg.Level.zapLevel())
}

func makeFileOutput(cfg Config) (zapcore.Core, error) {
	filepath := cfg.Files.Path
	name := cfg.Files.Name
	fullname := path.Join(filepath, name)

	r := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fullname,
		MaxSize:    int(cfg.Files.MaxSize / 1024 / 1024),
		MaxBackups: int(cfg.Files.MaxBackups),
	})

	return zapcore.NewCore(buildEncoder(cfg), r, cfg.Level.zapLevel()), nil
}

func globalLogger() *zap.Logger {
	return loadLogger().globalLogger
}

func loadLogger() *coreLogger {
	p := atomic.LoadPointer(&_log)
	return (*coreLogger)(p)
}

func storeLogger(l *coreLogger) {
	if old := loadLogger(); old != nil {
		old.rootLogger.Sync()
	}
	atomic.StorePointer(&_log, unsafe.Pointer(l))
}
