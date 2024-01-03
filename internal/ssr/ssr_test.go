package ssr

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestReplaceCustomElements(t *testing.T) {
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		reader  io.Reader
		wantErr bool
	}{
		{name: "replaces primary button inside as", reader: strings.NewReader(`<button as="button-primary">Click me</button>`), wantErr: false},
		{name: "replaces primary button element", reader: strings.NewReader(`<button-primary class="test">Click me</button-primary>`), wantErr: false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReplaceCustomElements(os.Stdout, tt.reader); (err != nil) != tt.wantErr {
				t.Errorf("ReplaceCustomElements() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
