package main

import (
	"fmt"
	"log"
	"sync"
)

type logger interface {
	newSubLogger(prefix string) logger

	Printf(fmt string, v ...any)
}

type rootLogger struct {
	lock *sync.Mutex
}

func newLogger() logger {
	return &rootLogger{
		lock: &sync.Mutex{},
	}
}

func (l *rootLogger) newSubLogger(prefix string) logger {
	return &subLogger{
		parent: l,
		prefix: prefix,
	}
}

func (l *rootLogger) Printf(fmt string, v ...any) {
	l.lock.Lock()
	defer l.lock.Unlock()

	log.Printf(fmt, v...)
}

type subLogger struct {
	parent logger
	prefix string
}

func (s *subLogger) newSubLogger(prefix string) logger {
	return &subLogger{
		parent: s,
		prefix: prefix,
	}
}

func (s *subLogger) Printf(format string, v ...any) {
	str := fmt.Sprintf(format, v...)
	s.parent.Printf("%s: %s", s.prefix, str)
}
