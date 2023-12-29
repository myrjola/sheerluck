package img

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"image/png"
	"os"
	"strings"
)

var Group = &cobra.Group{
	ID:    "img",
	Title: "Image operations",
}

func init() {
	Generate.Flags().String("out", "./out.png", "path to generated image file")
}

var Generate = &cobra.Command{
	Use:     "gen [prompt]",
	GroupID: "img",
	Short:   "Generate image",
	Long:    `Generates image with Dall-E`,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		ctx := context.Background()

		prompt := strings.Join(args, " ")

		request := openai.ImageRequest{
			Model:          openai.CreateImageModelDallE3,
			Prompt:         prompt,
			Size:           openai.CreateImageSize1024x1024,
			ResponseFormat: openai.CreateImageResponseFormatB64JSON,
			N:              1,
		}

		response, err := c.CreateImage(ctx, request)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Image creation error: %v\n", err)
			return
		}

		imgBytes, err := base64.StdEncoding.DecodeString(response.Data[0].B64JSON)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Base64 decode error: %v\n", err)
			return
		}

		r := bytes.NewReader(imgBytes)
		imgData, err := png.Decode(r)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PNG decode error: %v\n", err)
			return
		}

		outPath, err := cmd.Flags().GetString("out")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "invalid out flag: %v\n", err)
			return
		}
		file, err := os.Create(outPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "File creation error: %v\n", err)
			return
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		if err := png.Encode(file, imgData); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PNG encode error: %v\n", err)
			return
		}

		fmt.Printf("The image was saved as %s\n", outPath)
	},
}
