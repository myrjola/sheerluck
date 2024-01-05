package ssr

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"io"
	"slices"
	"strings"
)

func shortUID() string {
	b := make([]byte, 4) //equals 8 characters
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Errorf("rand.Read: %w", err))
	}
	return hex.EncodeToString(b)
}

//go:embed my-counter.webc
var mcf []byte

func ReplaceCustomElements(writer io.Writer, reader io.Reader) error {
	// TODO: add webc-files to ui folder
	// TODO: add bundler step to output webc-files scripts and styles to bundle.css and bundle.js files
	//       this should work with go generate
	// TODO: add bundles to the base template
	// TODO: add nonce support to middleware
	// TODO: add slots support
	// TODO: make ReplaceCustomElements support multiple webc-files (maybe I need a receiver function with initialiser)
	// TODO: JS and CSS bundler (probably needs to change the API so that I get them separately).
	//       One problem with this is that I can't push these to the base template, but maybe I can run the bundler
	//       at startup and include all the things required by components to their own files.
	//       Go generate would also work.

	myCounter, err := goquery.NewDocumentFromReader(bytes.NewReader(mcf))
	if err != nil {
		return fmt.Errorf("parse my-counter: %w", err)
	}
	myCounter.Find("script").Remove()
	myCounter.Find("style").Remove()

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return fmt.Errorf("parse reader: %w", err)
	}

	doc.Find("my-counter").Each(func(_ int, s *goquery.Selection) {
		props := make(map[string]string)
		// Parse the props
		for _, a := range s.Nodes[0].Attr {
			if strings.HasPrefix(a.Key, "@") {
				props[a.Key[1:]] = a.Val
			}
		}

		// Delete props from the node
		for k := range props {
			s.RemoveAttr("@" + k)
		}

		// Add uid prop
		props["uid"] = shortUID()

		mc := myCounter.Clone()
		for _, n := range mc.Nodes {
			// Replace all props as attributes inside the component
			var f func(*html.Node)
			f = func(n *html.Node) {
				htmlIndex := -1
				for i, attr := range n.Attr {
					if strings.HasPrefix(attr.Key, ":") {
						propKey := attr.Val
						// TODO: error handling
						if val, ok := props[propKey]; ok {
							n.Attr[i].Key = attr.Key[1:]
							n.Attr[i].Val = val
						}
					}

					// Insert inner HTML
					if attr.Key == "@html" {
						htmlIndex = i
						propKey := attr.Val
						// TODO: error handling
						if val, ok := props[propKey]; ok {
							n.AppendChild(&html.Node{
								Type: html.TextNode,
								Data: val,
							})
						}
					}
				}

				// Remove @html attrs
				if htmlIndex >= 0 {
					n.Attr = slices.Delete(n.Attr, htmlIndex, htmlIndex+1)
				}

				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(n)
		}

		// TODO: error handling
		myCounterHTML, _ := mc.Find("body").Html()
		s.SetHtml(myCounterHTML)
	})

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

	// Write out whole document if this is a base template
	if doc.Nodes[0].FirstChild.Type == html.DoctypeNode {
		return goquery.Render(writer, doc.Selection)
	}

	// Write out the innerhtml of body because the Golang html parser insists on adding it.
	// TODO: handle base template, maybe I can detect doctype to render the whole thing
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
