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
	Use:     "custom-elements",
	GroupID: "bundle",
	Short:   "Run bundler",
	Long:    "Bundles bundle.js and bundle.css files for custom elements",
	Run: func(cmd *cobra.Command, _ []string) {
		/*		bundler := components.NewBundler()
				bundler.AddCustomElement(components.CounterElement)
				err := bundler.Bundle()
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Bundler error: %v\n", err)
					return
				}
		*/

		fmt.Println("Bundling custom elements...")
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{"./ui/components/custom-elements.ts"},
			Bundle:      true,
			Outfile:     "./ui/static/bundle.js",
			Write:       true,
		})
		fmt.Println("Bundling complete!")

		if len(result.Errors) > 0 {
			fmt.Println("Bundler errors: %s", result.Errors[0])
			os.Exit(1)
		}
	},
}
