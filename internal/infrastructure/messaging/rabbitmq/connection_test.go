package rabbitmq

import (
	"strings"
	"testing"
)

func TestDefaultConfigURLFromEnvironment(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://svc:secret@broker:5672/")
	if got := DefaultConfig().URL; got != "amqp://svc:secret@broker:5672/" {
		t.Fatalf("URL = %q, want value from RABBITMQ_URL", got)
	}
}

func TestDefaultConfigURLCarriesNoEmbeddedCredentials(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")
	got := DefaultConfig().URL
	if strings.Contains(got, "@") {
		t.Fatalf("default URL %q must not embed credentials", got)
	}
}
