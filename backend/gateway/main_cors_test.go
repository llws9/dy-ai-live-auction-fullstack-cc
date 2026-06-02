package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCORSAllowHeadersIncludesIdempotencyKey(t *testing.T) {
	assert.Contains(t, defaultCORSAllowHeaders(), "X-Idempotency-Key")
}
