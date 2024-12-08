package main

import (
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
	"net/http"
)

type investigateTargetTemplateData struct {
	BaseTemplateData

	Investigation models.Investigation
}

func (app *application) investigateTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := contexthelpers.AuthenticatedUserID(ctx)
	investigationTargetID := r.PathValue("investigationTargetID")
	investigation, err := app.investigations.Get(ctx, investigationTargetID, userID)
	if err != nil {
		app.serverError(w, r, errors.Wrap(
			err,
			"get investigation",
			slog.String("investigation_target_id", investigationTargetID),
		))
		return
	}
	data := investigateTargetTemplateData{
		BaseTemplateData: newBaseTemplateData(r),
		Investigation:    *investigation,
	}
	app.render(w, r, http.StatusOK, "investigatetarget", data)
}
