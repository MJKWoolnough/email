package email

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

type Sender struct {
	send  chan sendEmail
	close chan struct{}
}

func NewSender(auth smtp.Auth, host, from string, timeout time.Duration) (*Sender, error) {
	if !strings.HasPrefix(host, "smtp://") && !strings.HasPrefix(host, "smtps://") {
		host = "smtp://" + host
	}
	address, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if address.Port() == "" {
		switch address.Scheme {
		case "smtp":
			address.Host += ":smtp"
		case "smtps":
			address.Host += ":465"
		}
	}
	encrypted := address.Scheme == "smtps"
	serverName := address.Hostname()
	var conn net.Conn
	if encrypted {
		conn, err = tls.Dial("tcp", address.Host, nil)
	} else {
		conn, err = net.Dial("tcp", address.Host)
	}
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return nil, err
	}
	if hasTLS, _ := client.Extension("STARTTLS"); hasTLS {
		if err = client.StartTLS(&tls.Config{ServerName: serverName}); err != nil {
			return nil, err
		}
	}
	if err = client.Auth(auth); err != nil {
		return nil, err
	}
	if err = client.Quit(); err != nil {
		return nil, err
	}
	s := &Sender{
		send:  make(chan sendEmail),
		close: make(chan struct{}),
	}
	go s.run(auth, serverName, address.Host, from, encrypted, timeout)
	return s
}

type sendEmail struct {
	to   string
	data Message
}

func (s *Sender) Send(to string, data Message) {
	s.send <- sendEmail{to, data}
}

func (s *Sender) Stop() {
	s.close <- struct{}{}
	<-s.close
}
