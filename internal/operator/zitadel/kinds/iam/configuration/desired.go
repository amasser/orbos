package configuration

import "github.com/caos/orbos/internal/secret"

type Configuration struct {
	Tracing                *Tracing       `yaml:"tracing,omitempty"`
	Cache                  *Cache         `yaml:"cache,omitempty"`
	Secrets                *Secrets       `yaml:"secrets,omitempty"`
	Notifications          *Notifications `yaml:"notifications,omitempty"`
	ConsoleEnvironmentJSON *secret.Secret `yaml:"consoleEnvironmentJSON,omitempty"`
}

type Secrets struct {
	Keys               *secret.Secret `yaml:"keys,omitempty"`
	UserVerificationID string         `yaml:"userVerificationID,omitempty"`
	OTPVerificationID  string         `yaml:"otpVerificationID,omitempty"`
	OIDCKeysID         string         `yaml:"oidcKeysID,omitempty"`
	CookieID           string         `yaml:"cookieID,omitempty"`
	CSRFID             string         `yaml:"csrfID,omitempty"`
}

type Notifications struct {
	GoogleChatURL *secret.Secret `yaml:"googleChatURL,omitempty"`
	Email         *Email         `yaml:"email,omitempty"`
	Twilio        *Twilio        `yaml:"twilio,omitempty"`
}

type Tracing struct {
	ServiceAccountJSON *secret.Secret `yaml:"serviceAccountJSON,omitempty"`
	ProjectID          string         `yaml:"projectID,omitempty"`
	Fraction           string         `yaml:"fraction,omitempty"`
}

type Twilio struct {
	SenderName string         `yaml:"senderName,omitempty"`
	AuthToken  *secret.Secret `yaml:"authToken,omitempty"`
	SID        *secret.Secret `yaml:"sid,omitempty"`
}

type Email struct {
	SMTPHost      string         `yaml:"smtpHost,omitempty"`
	SMTPUser      string         `yaml:"smtpUser,omitempty"`
	SenderAddress string         `yaml:"senderAddress,omitempty"`
	SenderName    string         `yaml:"senderName,omitempty"`
	TLS           bool           `yaml:"tls,omitempty"`
	AppKey        *secret.Secret `yaml:"appKey,omitempty"`
}

type Cache struct {
	MaxAge            string `yaml:"maxAge,omitempty"`
	SharedMaxAge      string `yaml:"sharedMaxAge,omitempty"`
	ShortMaxAge       string `yaml:"shortMaxAge,omitempty"`
	ShortSharedMaxAge string `yaml:"shortSharedMaxAge,omitempty"`
}
