package templates

import (
	"github.com/myrjola/sheerluck/internal/models"
	"io"
)

type PageTemplate interface {
	Render(ctx, w io.Writer, investigation models.Investigation) error
}
