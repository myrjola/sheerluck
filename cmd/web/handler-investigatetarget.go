package main

import (
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
	"net/http"
	"time"
)

type investigateTargetTemplateData struct {
	BaseTemplateData

	Investigation models.Investigation
}

func (app *application) investigateTargetGET(w http.ResponseWriter, r *http.Request) {
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

func (app *application) investigateTargetPOST(w http.ResponseWriter, r *http.Request) {
	//ctx := r.Context()
	//userID := contexthelpers.AuthenticatedUserID(ctx)
	//investigationTargetID := r.PathValue("investigationTargetID")
	//investigation, err := app.investigations.Get(ctx, investigationTargetID, userID)
	//if err != nil {
	//	app.serverError(w, r, errors.Wrap(
	//		err,
	//		"get investigation",
	//		slog.String("investigation_target_id", investigationTargetID),
	//	))
	//	return
	//}

	ctx := r.Context()
	ch := make(chan string, 1)
	go func() {
		time.Sleep(10000 * time.Second) //nolint:mnd // 10000 seconds should be enough to trigger timeout.
		ch <- "done"
	}()

	select {
	case <-ctx.Done():
		err := errors.Wrap(ctx.Err(), "wait for investigation")
		app.logger.LogAttrs(ctx, slog.LevelInfo, "Context cancelled", errors.SlogError(err))
	case result := <-ch:
		app.logger.Info("Investigation done", slog.String("result", result))
	}

	http.Redirect(w, r, r.RequestURI, http.StatusSeeOther)
}
