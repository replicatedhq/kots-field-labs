package main

import (
	"encoding/json"
	"github.com/replicatedhq/kots-field-labs/setup/pkg/fieldlabs"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		params   fieldlabs.Params
		files    map[string]string
		wantEnvs []fieldlabs.Environment
		wantLabs []fieldlabs.Environment
		wantErr  string
	}{
		{
			name: "csv",
			files: map[string]string{
				"testing-envs.csv": `Timestamp,name,email,slug,pub_key,password
4/9/2021 23:07:06,Dex,dex@replicated.com,dex,ssh-rsa public-key-bytes dex@replicated.com,password`,
				"testing-labs.json": `[]`,
			},
			params: fieldlabs.Params{
				EnvironmentsCSV: "testing-envs.csv",
				LabsJSON:        "testing-labs.json",
			},
			wantEnvs: []fieldlabs.Environment{
				{
					Name:            "Dex",
					Slug:            "dex",
					PubKey:          "ssh-rsa public-key-bytes dex@replicated.com",
					Email:           "dex@replicated.com",
					KotsadmPassword: "password",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			for name, contents := range test.files {
				err := ioutil.WriteFile(name, []byte(contents), 0644)
				req.NoError(err)
			}

			defer func() {
				for name := range test.files {
					_ = os.RemoveAll(name)
				}
			}()
			envs, labs, err := loadConfig(&test.params)
			if test.wantErr != "" {
				req.EqualError(err, test.wantErr)
			}

			if test.wantEnvs != nil {
				req.JSONEq(mustJSON(test.wantEnvs), mustJSON(envs))
			}
			if test.wantLabs != nil {
				req.JSONEq(mustJSON(test.wantLabs), mustJSON(labs))
			}

		})
	}
}

func mustJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bytes)

}
