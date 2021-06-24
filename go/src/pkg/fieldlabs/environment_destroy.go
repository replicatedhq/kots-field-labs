package fieldlabs

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/replicatedhq/replicated/pkg/types"
	"strings"
)

func (e *EnvironmentManager) Destroy(envs []Environment) error {
	var appsToDelete []types.App
	apps, err := e.GClient.ListApps()
	if err != nil {
		return errors.Wrapf(err, "list apps")
	}

	for _, env := range envs {
		testString := fmt.Sprintf("%s-%s", e.Params.NamePrefix, env.Slug)
		// find all apps matching the prefixed slug for this env
		for _, app := range apps {
			if strings.Contains(app.App.Slug, testString) {
				appsToDelete = append(appsToDelete, *app.App)
			}
		}
	}

	err = e.PrintApps(appsToDelete)
	if err != nil {
		return errors.Wrap(err, "print apps to delete")
	}
	// confirm delete
	answer, err := PromptConfirmDelete()
	if err != nil {
		return errors.Wrap(err, "confirm delete")
	}

	if answer != "yes" {
		return errors.New("prompt declined")
	}

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
