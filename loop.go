package email

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"time"
)

// runs in its own goroutine
func (s *Sender) run(auth smtp.Auth, serverName, host, from string, encrypted bool, timeout time.Duration) {
	var (
		timer  *time.Timer
		client *smtp.Client
		err    error
	)
	if timeout >= 0 {
		timer = time.NewTimer(time.Hour)
		timer.Stop()
	} else {
		timer = new(time.Timer)
	}
	tlsConfig := tls.Config{
		ServerName: serverName,
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
				var conn net.Conn
				if encrypted {
					conn, err = tls.Dial("tcp", host, &tlsConfig)
				} else {
					conn, err = net.Dial("tcp", host)
				}
				client, err = smtp.NewClient(conn, host)
				if err != nil {
					//TODO:handle
					continue
				}
				if !encrypted {
					if hasTLS, _ := client.Extension("STARTTLS"); hasTLS {
						err = client.StartTLS(&tlsConfig)
						if err != nil {
							client.Close()
							client = nil
							//TODO:handle
							continue
						}
					}
				}
				err = client.Auth(auth)
				if err != nil {
					client.Close()
					client = nil
					//TODO:handle
					continue
				}
			}

			err = client.Mail(from)
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
			se.data.MessageTo(wc)
			wc.Close()

			if timeout > 0 {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(timeout)
			} else if timeout == 0 {
				client.Close()
				client = nil
			}
		}
	}
}
