package nws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		temp           int
		classification string
	}{
		{105, "very hot"},
		{100, "very hot"},
		{99, "hot"},
		{85, "hot"},
		{84, "moderate"},
		{56, "moderate"},
		{55, "cold"},
		{31, "cold"},
		{30, "very cold"},
		{-10, "very cold"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.classification, classify(tc.temp), "classify(%d)", tc.temp)
	}
}
