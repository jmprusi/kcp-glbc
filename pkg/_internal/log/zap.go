/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"flag"
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EncoderConfigOption is a function that can modify a `zapcore.EncoderConfig`.
type EncoderConfigOption func(*zapcore.EncoderConfig)

// NewEncoderFunc is a function that creates an Encoder using the provided EncoderConfigOptions.
type NewEncoderFunc func(...EncoderConfigOption) zapcore.Encoder

// New returns a brand new Logger configured with Opts. It
// uses KcpAwareEncoder which adds Type information and
// Namespace/Name to the log.
func New(opts ...Opts) logr.Logger {
	return zapr.NewLogger(NewRaw(opts...))
}

// Opts allows to manipulate Options.
type Opts func(*Options)

// WriteTo configures the logger to write to the given io.Writer, instead of standard error.
// See Options.DestWriter.
func WriteTo(out io.Writer) Opts {
	return func(o *Options) {
		o.DestWriter = out
	}
}

// Encoder configures how the logger will encode the output e.g JSON or console.
// See Options.Encoder.
func Encoder(encoder zapcore.Encoder) func(o *Options) {
	return func(o *Options) {
		o.Encoder = encoder
	}
}

func newJSONEncoder(opts ...EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func newConsoleEncoder(opts ...EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// Level sets Options.Level, which configures the minimum enabled logging level e.g. Debug, Info.
// A zap log level should be multiplied by -1 to get the logr verbosity.
// For example, to get logr verbosity of 3, pass zapcore.Level(-3) to this Opts.
// See https://pkg.go.dev/github.com/go-logr/zapr for how zap level relates to logr verbosity.
func Level(level zapcore.LevelEnabler) func(o *Options) {
	return func(o *Options) {
		o.Level = level
	}
}

// Options contains all possible settings.
type Options struct {
	// Encoder configures how Zap will encode the output.  Defaults to
	// console when Development is true and JSON otherwise
	Encoder zapcore.Encoder
	// EncoderConfigOptions can modify the EncoderConfig needed to initialize an Encoder.
	// See https://godoc.org/go.uber.org/zap/zapcore#EncoderConfig for the list of options
	// that can be configured.
	// Note that the EncoderConfigOptions are not applied when the Encoder option is already set.
	EncoderConfigOptions []EncoderConfigOption
	// NewEncoder configures Encoder using the provided EncoderConfigOptions.
	// Note that the NewEncoder function is not used when the Encoder option is already set.
	NewEncoder NewEncoderFunc
	// DestWriter controls the destination of the log output.  Defaults to
	// os.Stderr.
	DestWriter io.Writer
	// Level configures the verbosity of the logging.
	// Defaults to Debug when Development is true and Info otherwise.
	// A zap log level should be multiplied by -1 to get the logr verbosity.
	// For example, to get logr verbosity of 3, set this field to zapcore.Level(-3).
	// See https://pkg.go.dev/github.com/go-logr/zapr for how zap level relates to logr verbosity.
	Level zapcore.LevelEnabler
	// StacktraceLevel is the level at and above which stacktraces will
	// be recorded for all messages. Defaults to Warn when Development
	// is true and Error otherwise.
	// See Level for the relationship of zap log level to logr verbosity.
	StacktraceLevel zapcore.LevelEnabler
	// ZapOpts allows passing arbitrary zap.Options to configure on the
	// underlying Zap logger.
	ZapOpts []zap.Option
}

// addDefaults adds defaults to the Options.
func (o *Options) addDefaults() {
	if o.DestWriter == nil {
		o.DestWriter = os.Stderr
	}
	if o.NewEncoder == nil {
		o.NewEncoder = newConsoleEncoder
	}
	if o.Level == nil {
		lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
		o.Level = &lvl
	}
	if o.StacktraceLevel == nil {
		lvl := zap.NewAtomicLevelAt(zap.PanicLevel)
		o.StacktraceLevel = &lvl
	}
	if o.Encoder == nil {
		o.Encoder = o.NewEncoder(o.EncoderConfigOptions...)
	}
	o.ZapOpts = append(o.ZapOpts, zap.AddStacktrace(o.StacktraceLevel))
}

// NewRaw returns a new zap.Logger configured with the passed Opts
// or their defaults. It uses KcpAwareEncoder which adds Type
// information and Namespace/Name to the log.
func NewRaw(opts ...Opts) *zap.Logger {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	o.addDefaults()

	// this basically mimics New<type>Config, but with a custom sink
	sink := zapcore.AddSync(o.DestWriter)

	o.ZapOpts = append(o.ZapOpts, zap.AddCallerSkip(1), zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(&KcpAwareEncoder{Encoder: o.Encoder, Verbose: false}, sink, o.Level))
	log = log.WithOptions(o.ZapOpts...)
	return log
}

// BindFlags will parse the given flagset for zap option flags and set the log options accordingly
// zap-encoder: Zap log encoding (one of 'json' or 'console')
// zap-log-level:  Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error',
// or any integer value > 0 which corresponds to custom debug levels of increasing verbosity")
// zap-stacktrace-level: Zap Level at and above which stacktraces are captured (one of 'info', 'error' or 'panic')
func (o *Options) BindFlags(fs *flag.FlagSet) {
	// Set Encoder value
	var encVal encoderFlag
	encVal.setFunc = func(fromFlag NewEncoderFunc) {
		o.NewEncoder = fromFlag
	}
	fs.Var(&encVal, "zap-encoder", "Zap log encoding (one of 'json' or 'console')")

	// Set the Log Level
	var levelVal levelFlag
	levelVal.setFunc = func(fromFlag zapcore.LevelEnabler) {
		o.Level = fromFlag
	}
	fs.Var(&levelVal, "zap-log-level",
		"Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', "+
			"or any integer value > 0 which corresponds to custom debug levels of increasing verbosity")

	// Set the StackTrace Level
	var stackVal stackTraceFlag
	stackVal.setFunc = func(fromFlag zapcore.LevelEnabler) {
		o.StacktraceLevel = fromFlag
	}
	fs.Var(&stackVal, "zap-stacktrace-level",
		"Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').")
}

// UseFlagOptions configures the logger to use the Options set by parsing zap option flags from the CLI.
// opts := zap.Options{}
// opts.BindFlags(flag.CommandLine)
// flag.Parse()
// log := zap.New(zap.UseFlagOptions(&opts))
func UseFlagOptions(in *Options) Opts {
	return func(o *Options) {
		*o = *in
	}
}
