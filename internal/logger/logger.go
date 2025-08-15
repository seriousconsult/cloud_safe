package logger

import (
        "log"
        "os"
)

type Logger struct {
        verbose bool
        info    *log.Logger
        error   *log.Logger
        debug   *log.Logger
}

func New(verbose bool) *Logger {
        return &Logger{
                verbose: verbose,
                info:    log.New(os.Stdout, "[INFO] ", log.LstdFlags),
                error:   log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
                debug:   log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
        }
}

func (l *Logger) Info(msg string) {
        l.info.Println(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
        l.info.Printf(format, args...)
}

func (l *Logger) Error(msg string) {
        l.error.Println(msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
        l.error.Printf(format, args...)
}

func (l *Logger) Debug(msg string) {
        if l.verbose {
                l.debug.Println(msg)
        }
}

func (l *Logger) Debugf(format string, args ...interface{}) {
        if l.verbose {
                l.debug.Printf(format, args...)
        }
}

func (l *Logger) Fatal(msg string) {
        l.error.Println(msg)
        os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
        l.error.Printf(format, args...)
        os.Exit(1)
}
