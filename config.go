// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package logp

import "time"

// Config contains the configuration options for the logger. To create a Config
// from a common.Config use logp/config.Build.
type Config struct {
	JSON      bool     `mapstructure:"json" json:"json"`           // Write logs as JSON.
	Level     Level    `mapstructure:"level" json:"level"`         // Logging level (error, warning, info, debug).
	Selectors []string `mapstructure:"selectors" json:"selectors"` // Selectors for debug level logging.

	toObserver  bool
	toIODiscard bool
	ToStderr    bool `mapstructure:"to_stderr" json:"to_stderr"`
	ToSyslog    bool `mapstructure:"to_syslog" json:"to_syslog"`
	ToFiles     bool `mapstructure:"to_files" json:"to_files"`
	ToEventLog  bool `mapstructure:"to_eventlog" json:"to_event_log"`

	Files FileConfig `mapstructure:"files" json:"files"`

	addCaller   bool // Adds package and line number info to messages.
	development bool // Controls how DPanic behaves.
}

// FileConfig contains the configuration options for the file output.
type FileConfig struct {
	Path        string        `mapstructure:"path" json:"path"`
	Name        string        `mapstructure:"name" json:"name"`
	MaxSize     uint          `mapstructure:"rotateeverybytes" validate:"min=1" json:"max_size"`
	MaxBackups  uint          `mapstructure:"keepfiles" validate:"max=1024" json:"max_backups"`
	Permissions uint32        `mapstructure:"permissions" json:"permissions"`
	Interval    time.Duration `mapstructure:"interval" json:"interval"`
}

var defaultConfig = Config{
	Level:    InfoLevel,
	ToFiles:  false,
	ToStderr: true,
	Files: FileConfig{
		MaxSize:     10 * 1024 * 1024,
		MaxBackups:  7,
		Permissions: 0600,
		Interval:    0,
	},
	addCaller: true,
}

// DefaultConfig returns the default config options.
func DefaultConfig() Config {
	return defaultConfig
}

type Console struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type File struct {
	Name     string `json:"name"`
	FileName string `json:"file_name"`
}

type RollingFile struct {
	Name string

	// FileName is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	FileName string `json:"file_name"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `json:"max_size"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `json:"max_age"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"max_backups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"local_time"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress"`
}
