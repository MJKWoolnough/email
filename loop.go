package email

import (
	"crypto/tls"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// runs in its own goroutine
func (s *Sender) run(auth smtp.Auth, host, from string, timeout time.Duration) {
	if !strings.HasPrefix("smtp://") && !strings.HasPrefix("smtps://") {
		host = "smtp://" + host
	}
	address, _ := url.Parse(host)
	if address.Port() == "" {
		switch address.Scheme {
		case "smtp":
			address.Host += ":smtp"
		case "smtps":
			address.Host += ":465"
		}
	}
	serverName := address.Hostname()
	var (
		timer  *time.Timer
		client *smtp.Client
		err    error
	)
	if timeout > 0 {
		timer = time.NewTimer(time.Hour)
		timer.Stop()
	} else {
		timer = new(time.Timer)
	}
	for {
		select {
		case <-timer.C:
			client.Quit()
			client.Close()
			client = nil
		case <-s.close:
			if client != nil {
				client.Close()
				if !timer.Stop() {
					<-timer.C
				}
			}
			close(s.send)
			close(s.close)
			return
		case se := <-s.send:
			if client != nil && client.Noop() != nil {
				client.Close()
				client = nil
			}
			if client == nil {
				client, err = smtp.Dial(host)
				if err != nil {
					//TODO:handle
					continue
				}
				if hasTLS, _ := client.Extension("STARTTLS"); hasTLS {
					err = client.StartTLS(&tls.Config{ServerName: serverName})
					if err != nil {
						client.Close()
						client = nil
						//TODO:handle
						continue
					}
				}
				err = client.Auth(e.auth)
				if err != nil {
					client.Close()
					client = nil
					//TODO:handle
					continue
				}
			}

			err = client.Mail(e.from)
			if err != nil {
				client.Reset()
				//TODO:handle
				continue
			}

			err = client.Rcpt(se.to)
			if err != nil {
				client.Reset()
				//TODO:handle
				continue
			}

			wc, err := client.Data()
			if err != nil {
				client.Reset()
				//TODO:handle
				continue
			}
			_, err = se.data.WriteTo(wc)
			if err != nil {
				client.Reset()
				//TODO:handle
				continue
			}
			wc.Close()

			if e.timeout > 0 {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(e.timeout)
			}
		}
	}
}
