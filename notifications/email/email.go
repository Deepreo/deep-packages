/*
Copyright © 2024 Deepreo Siber Güvenlik A.S Resul ÇELİK <resul.celik@deepreo.com>
*/

package mailler

import (
	"bytes"
	"crypto/tls"
	"errors"
	"html/template"
	"io"
	"sync"

	"gopkg.in/gomail.v2"
)

type Config struct {
	Host     string     `mapstructure:"host"`
	Port     int        `mapstructure:"port"`
	Username string     `mapstructure:"username"`
	Password string     `mapstructure:"password"`
	Sender   MailSender `mapstructure:"sender"`
}

type MailSender struct {
	Address string `mapstructure:"address"`
	Name    string `mapstructure:"name"`
}

type mailTemplate struct {
	templateFilePath string
	data             interface{}
}

type mailRequest struct {
	subject  string
	sender   *MailSender
	mails    []string
	template *mailTemplate
	body     string
}

type Mailler struct {
	errChan       chan error
	wg            sync.WaitGroup
	dialler       *gomail.Dialer
	defaultSender *MailSender
	errChanSize   *int
	tlsConfig     *tls.Config
}

// SetTLSConfig sets the tls config for the mailer
func SetTLSConfig(tlsConfig *tls.Config) func(*Mailler) {
	return func(mailler *Mailler) {
		mailler.tlsConfig = tlsConfig
	}
}

// SetErrorChanSize sets the error channel size for the mailer
func SetErrorChanSize(size int) func(*Mailler) {
	return func(mailler *Mailler) {
		mailler.errChanSize = &size
	}
}

// NewMailler creates a new mailer
func NewMailler(cfg *Config, opts ...func(*Mailler)) (*Mailler, error) {
	m := new(Mailler)
	for _, opt := range opts {
		opt(m)
	}
	if cfg.Sender.Address == "" {
		return nil, errors.New("sender address not set")
	}
	if cfg.Sender.Name == "" {
		return nil, errors.New("sender name not set")
	}
	m.defaultSender = &cfg.Sender
	if m.errChanSize == nil {
		m.errChanSize = new(int)
		*m.errChanSize = 10
	}

	errchan := make(chan error, *m.errChanSize)
	dialler := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if m.tlsConfig != nil {
		dialler.TLSConfig = m.tlsConfig
	}
	if ping, err := dialler.Dial(); err != nil {
		return nil, errors.Join(errors.New("failed to connect to email server"), err)
	} else {
		ping.Close()
	}
	m.errChan = errchan
	m.dialler = dialler
	return m, nil
}

func (e *Mailler) Send(req *mailRequest) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		m := gomail.NewMessage()
		if req.sender != nil {
			m.SetHeader("From", m.FormatAddress(req.sender.Address, req.sender.Name))
		} else {
			m.SetHeader("From", m.FormatAddress(e.defaultSender.Address, e.defaultSender.Name))
		}
		if len(req.mails) > 1 {
			m.SetHeader("Bcc", req.mails...)
		} else {
			m.SetHeader("To", req.mails[0])
		}
		m.SetHeader("Subject", req.subject)
		m.SetBody("text/html", req.body)
		if err := e.dialler.DialAndSend(m); err != nil {
			e.errChan <- err
		}
	}()
}

func (e *Mailler) GetErrors() chan error {
	return e.errChan
}

func (e *Mailler) WriteErrors(writer io.Writer) {
	for err := range e.errChan {
		if err != nil {
			writer.Write([]byte(err.Error() + "\n"))
		}
	}
}

func (e *Mailler) WaitForCompletion() {
	e.wg.Wait()
	close(e.errChan)
}

func SetBodyWithTemplate(templateFilePath string, data interface{}) func(*mailRequest) {
	return func(m *mailRequest) {
		m.template = &mailTemplate{
			templateFilePath: templateFilePath,
			data:             data,
		}
	}
}

func SetBodyWithText(text string) func(*mailRequest) {
	return func(m *mailRequest) {
		m.body = text
	}
}

func SetSender(sender *MailSender) func(*mailRequest) {
	return func(m *mailRequest) {
		m.sender = sender
	}
}

func NewMail(subject string, mails []string, bodyOpts ...func(*mailRequest)) (*mailRequest, error) {
	m := new(mailRequest)
	m.subject = subject
	m.mails = mails
	for _, opt := range bodyOpts {
		opt(m)
	}
	if m.template != nil {
		if m.template.templateFilePath == "" {
			return nil, errors.New("template file path not set")
		}
		if m.template.data == nil {
			return nil, errors.New("template data not set")
		}
		parsed, err := parseMailTemplate(m.template.data, m.template.templateFilePath)
		if err != nil {
			return nil, err
		}
		m.body = parsed

	} else {
		if m.body == "" {
			return nil, errors.New("body not set")
		}
	}

	return m, nil
}

func parseMailTemplate(p interface{}, templatePath string) (string, error) {
	parsedTemplate, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", errors.Join(errors.New("failed to parse template"), err)
	}

	buf := new(bytes.Buffer)
	err = parsedTemplate.Execute(buf, p)
	if err != nil {
		return "", errors.Join(errors.New("failed to execute template"), err)
	}
	return buf.String(), nil
}
