package main

import (
	"testing"
)

func TestWGgetConfig(t *testing.T) {
	TESTING = true

	config := WGgetConfig()
	config2 := WGgetConfig()

	if config != config2 {
		t.Error("config and config2 are expected to be the same object")
	}
}
