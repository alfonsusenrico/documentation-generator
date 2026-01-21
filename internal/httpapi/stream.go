package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

type streamEvent struct {
	Type    string      `json:"type"`
	Message string      `json:"message,omitempty"`
	Detail  string      `json:"detail,omitempty"`
	Percent int         `json:"percent,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type streamWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newStreamWriter(w http.ResponseWriter) (*streamWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported")
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	return &streamWriter{w: w, flusher: flusher}, nil
}

func (s *streamWriter) Send(event streamEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := s.w.Write(append(b, '\n')); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

type progressTracker struct {
	total   int
	current int
	send    func(message, detail string, percent int)
}

func newProgressTracker(total int, send func(message, detail string, percent int)) *progressTracker {
	return &progressTracker{total: total, send: send}
}

func (p *progressTracker) Step(message, detail string) {
	if p == nil || p.total <= 0 || p.send == nil {
		return
	}
	if p.current < p.total {
		p.current++
	}
	percent := p.current * 100 / p.total
	if percent > 100 {
		percent = 100
	}
	p.send(message, detail, percent)
}
