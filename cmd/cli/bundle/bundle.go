package bundle

import (
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
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
	Use:     "bundle",
	GroupID: "bundle",
	Short:   "Run bundler",
	Long:    "Bundles bundle.js and bundle.css files.",
	Run: func(_ *cobra.Command, _ []string) {
		/*		bundler := components.NewBundler()
				bundler.AddCustomElement(components.CounterElement)
				err := bundler.Bundle()
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Bundler error: %v\n", err)
					return
				}
		*/

		fmt.Println("Bundling...")
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{"./main.ts"},
			Bundle:      true,
			Outfile:     "./ui/static/bundle.js",
			Write:       true,
		})
		if len(result.Errors) > 0 {
			fmt.Println("Bundler errors: ", result.Errors[0])
			os.Exit(1)
		}
		fmt.Println("Bundling complete!")
	},
}
