package observability

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu  sync.Mutex
	out *log.Logger
}

func NewLogger() *Logger { return &Logger{out: log.New(os.Stdout, "", 0)} }
func (l *Logger) Event(level, message string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	payload := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "message": message}
	for k, v := range fields {
		payload[k] = v
	}
	b, _ := json.Marshal(payload)
	l.out.Print(string(b))
}
func (l *Logger) Info(message string, fields map[string]any)  { l.Event("info", message, fields) }
func (l *Logger) Error(message string, fields map[string]any) { l.Event("error", message, fields) }
