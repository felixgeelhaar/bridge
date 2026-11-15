package rabbitmq

import (
	"strings"
	"testing"
)

func TestDefaultConfigURLFromEnvironment(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "url-from-environment")
	if got := DefaultConfig().URL; got != "url-from-environment" {
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
