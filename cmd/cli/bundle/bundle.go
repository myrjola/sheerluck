package bundle

import (
	"fmt"
	"github.com/myrjola/sheerluck/ui/components"
	"github.com/spf13/cobra"
	"os"
)

var Group = &cobra.Group{
	ID:    "bundle",
	Title: "Bundler",
}

func init() {
}

var CustomElements = &cobra.Command{
	Use:     "custom-elements",
	GroupID: "bundle",
	Short:   "Run bundler",
	Long:    "Bundles bundle.js and bundle.css files for custom elements",
	Run: func(cmd *cobra.Command, _ []string) {
		bundler := components.NewBundler()
		bundler.AddCustomElement(components.CounterElement)
		err := bundler.Bundle()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Bundler error: %v\n", err)
			return
		}
	},
}
