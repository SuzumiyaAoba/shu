package app

import (
	"bytes"
	"testing"
)

func TestOpenInMemory(t *testing.T) {
	instance, err := Open(Config{
		DBPath:    ":memory:",
		LogLevel:  "info",
		LogOutput: new(bytes.Buffer),
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	if instance.Service == nil {
		t.Fatal("expected Service to be initialized")
	}
	if instance.Close == nil {
		t.Fatal("expected Close to be initialized")
	}
}

func TestOpenInvalidLogLevel(t *testing.T) {
	_, err := Open(Config{
		DBPath:    ":memory:",
		LogLevel:  "trace",
		LogOutput: new(bytes.Buffer),
	})
	if err == nil {
		t.Fatal("expected invalid log level error")
	}
}
