package mailer

import (
	"bytes"
	"embed"
	"github.com/go-mail/mail/v2"
	"html/template"
	"time"
)

// Declare a new variable with the type embed.FS (embedded file system) to hold email
// templates. This has a comment directive in the format `//go:embed <path>` IMMEDIATELY
// ABOVE it, which indicates to Go that we want to store the contents of the ./templates
// directory in the templateFS embedded file system variable.

//go:embed "templates"
var templateFS embed.FS

// Define a Mailer struct which contains a mail.Dialer instance (used to connect to a
// SMTP server) and the sender information for your emails.
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// Initialize a new Mailer instance with the given SMTP server settings. We
// also configure this to use a 5-second timeout whenever we send an email.
func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// The Send method takes the recipient email address as the first parameter,
// the name of the file containing the templates, and any dynamic data for
// the templates as an interface{} parameter.
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {

	// Parse the required template file from the embedded file system.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the named template "subject", passing in the dynamic data.
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// Follow the same pattern to execute the "plainBody" template.
	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	// And likewise with the "htmlBody" template.
	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Initialize a new mail.Message instance. Set relevant mail headers, use the SetBody
	// method to set the plain-text body, the AddAlternative method to set the HTML body.
	// It's important to note that AddAlternative() should always be called after SetBody.
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Open a connection to the SMTP server, send the message, then close the connection.
	return m.dialer.DialAndSend(msg)
}
