package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_computeDestinationFolder(t *testing.T) {
	t.Run("verify destination folder", func(t *testing.T) {
		result := computeDestinationFolder("D:/Ali/OneDrive/Images", time.Date(2024, 3, 6, 15, 22, 12, 1, time.UTC))
		assert.Equal(t, "D:/Ali/OneDrive/Images/2024/2024-03-06", result)
	})
}
