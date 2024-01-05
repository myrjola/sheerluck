package components

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
	"os"
	"strings"
)

func shortUID() string {
	b := make([]byte, 4) // equals 8 characters
	rand.Read(b)
	return hex.EncodeToString(b)
}

type CustomElement struct {
	Script string
	Style  string
}

type Bundler struct {
	customElements []CustomElement
}

func NewBundler() *Bundler {
	return &Bundler{}
}

func (b *Bundler) AddCustomElement(ce CustomElement) {
	b.customElements = append(b.customElements, ce)
}

func (b *Bundler) Bundle() error {
	// concatenate scripts and styles
	scriptBuilder := strings.Builder{}
	styleBuilder := strings.Builder{}
	for _, ce := range b.customElements {
		scriptBuilder.WriteString(ce.Script)
		styleBuilder.WriteString(ce.Style)
	}

	// minify scripts
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)
	minifiedScripts, err := m.String("text/javascript", scriptBuilder.String())

	// write scripts and styles to files
	scriptF, err := os.Create("./ui/static/bundle.js")
	if err != nil {
		return fmt.Errorf("create bundle.js: %w", err)
	}
	defer scriptF.Close()
	_, err = scriptF.WriteString(minifiedScripts)
	if err != nil {
		return fmt.Errorf("write bundle.js: %w", err)
	}

	styleF, err := os.Create("./bundle.css")
	if err != nil {
		return fmt.Errorf("create bundle.css: %w", err)
	}
	defer styleF.Close()
	_, err = styleF.WriteString(styleBuilder.String())
	if err != nil {
		return fmt.Errorf("write bundle.css: %w", err)
	}
	return nil
}
