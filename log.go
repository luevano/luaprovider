package luaprovider

import (
	"fmt"
	"sync"

	"github.com/luevano/libmangal/logger"
)

// providerLogger logs through the libmangal logger with the provider ID as prefix, one per provider.
//
// The libmangal logger is set by libmangal (mangal plugs into it for info display), messages logged
// before it's set are kept and logged once it is.
type providerLogger struct {
	mu      sync.Mutex
	prefix  string
	logger  *logger.Logger
	backlog []string
}

func newProviderLogger(prefix string) *providerLogger {
	return &providerLogger{prefix: prefix}
}

// setLogger sets the libmangal logger and logs the messages kept until now.
func (l *providerLogger) setLogger(logger *logger.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger = logger
	for _, msg := range l.backlog {
		l.logger.Log("[backlog] %s", msg)
	}
	l.backlog = nil
}

func (l *providerLogger) log(format string, a ...any) {
	msg := l.prefix + fmt.Sprintf(format, a...)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logger == nil {
		l.backlog = append(l.backlog, msg)
		return
	}
	l.logger.Log("%s", msg)
}
