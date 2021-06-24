package main

import (
	"encoding/json"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-field-labs/setup/pkg/fieldlabs"
	"io/ioutil"
	"os"
)

func loadConfig(params *fieldlabs.Params) ([]fieldlabs.Environment, []fieldlabs.LabSpec, error) {
	environments := []fieldlabs.Environment{}
	if params.EnvironmentsJSON != "" {
		envJSON, err := ioutil.ReadFile(params.EnvironmentsJSON)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "read environments json from %q", params.EnvironmentsJSON)
		}
		err = json.Unmarshal(envJSON, &environments)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "read labs json from %q", params.EnvironmentsJSON)
		}
	} else if params.EnvironmentsCSV != "" {
		envCSV, err := os.Open(params.EnvironmentsCSV)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "read environments csv from %q", params.EnvironmentsJSON)
		}
		err = gocsv.Unmarshal(envCSV, &environments)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "read labs csv from %q", params.EnvironmentsJSON)
		}

	} else {
		return nil, nil, missingParam("exactly one of REPLICATED_ENVIRONMENTS_JSON or REPLICATED_ENVIRONMENTS_CSV")
	}

	labs := []fieldlabs.LabSpec{}
	labJSON, err := ioutil.ReadFile(params.LabsJSON)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "read labs json from %q", params.LabsJSON)
	}
	err = json.Unmarshal(labJSON, &labs)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshal labs json from %q", params.LabsJSON)
	}

	return environments, labs, nil
}
