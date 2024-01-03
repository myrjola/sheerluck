package ssr

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"io"
)

func ReplaceCustomElements(writer io.Writer, reader io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	// buttonClass := "rounded-md bg-indigo-800 text-sm font-semibold text-white shadow-sm px-3.5 py-2.5 hover:bg-indigo-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-400"
	buttonClass := "bg-indigo-800"
	doc.Find("button-primary").Each(func(_ int, s *goquery.Selection) {
		s.AddClass(buttonClass)
	})
	doc.Find(`[as="button-primary"]`).Each(func(_ int, s *goquery.Selection) {
		s.RemoveAttr("as")
		s.AddClass(buttonClass)
		nodes := s.Nodes
		nodes[0].Data = "button-primary"
	})

	// Third pass: recover the gohtml templating
	body := doc.Find("body")
	if len(body.Nodes) > 0 {
		for c := body.Nodes[0].FirstChild; c != nil; c = c.NextSibling {
			if err = html.Render(writer, c); err != nil {
				return fmt.Errorf("render html: %w", err)
			}
		}
	}
	return nil
}
