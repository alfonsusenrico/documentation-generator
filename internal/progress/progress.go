package progress

type Event struct {
	Message string
	Detail  string
}

type Reporter interface {
	Report(Event)
}

type ReporterFunc func(Event)

func (f ReporterFunc) Report(e Event) {
	if f != nil {
		f(e)
	}
}
