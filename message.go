package email

import "io"

type Message interface {
	MessageTo(io.Writer)
}

type MessageBytes []byte

func (m MessageBytes) MessageTo(w io.Writer) {
	w.Write(m)
}

type MessageString string

func (m MessageString) MessageTo(w io.Writer) {
	w.Write([]byte(m))
}

type Template interface {
	Execute(io.Writer, interface{})
}

type MessageTemplate struct {
	Template Template
	Data     interface{}
}

func (m MessageTemplate) MessageTo(w io.Writer) {
	m.Template.Execute(w, m.Data)
}
