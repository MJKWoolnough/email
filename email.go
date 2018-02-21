package email

import (
	"io"
	"net/smtp"
	"time"
)

type Sender struct {
	send  chan sendEmail
	close chan struct{}
}

func NewSender(auth smtp.Auth, address, from string, timeout time.Duration) *Sender {
	s := &Sender{
		send:  make(chan sendEmail),
		close: make(chan struct{}),
	}
	go s.run(auth, address, from, timeout)
	return s
}

type sendEmail struct {
	to   string
	data io.WriterTo
}

func (s *Sender) Send(to string, data io.WriterTo) {
	s.send <- sendEmail{to, data}
}

func (s *Sender) Stop() {
	s.close <- struct{}{}
	<-s.close
}
