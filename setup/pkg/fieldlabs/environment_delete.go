package fieldlabs

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/replicated/pkg/types"
)

func (e *EnvironmentManager) deleteApps(appsToDelete []types.App) error {
	for _, app := range appsToDelete {
		e.Log.ActionWithSpinner(fmt.Sprintf("Deleting App %s", app.Slug))
		err := e.GClient.DeleteKOTSApp(app.ID)
		if err != nil {
			e.Log.FinishSpinnerWithError()
			return errors.Wrapf(err, "delete app %q %q", app.Slug, app.ID)
		}
		e.Log.FinishSpinner()
	}

	return nil
}
