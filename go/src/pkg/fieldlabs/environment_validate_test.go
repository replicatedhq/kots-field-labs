package fieldlabs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantErr string
	}{
		{
			name: "basic",
			env: Environment{
				Name:            "Dex",
				Slug:            "dex",
				KotsadmPassword: "password",
			},
		},
		{
			name: "no slug",
			env: Environment{
				Name: "Dex",
			},
			wantErr: "no slug set for env Dex",
		},
		{
			name: "bad slug",
			env: Environment{
				Name: "Dex",
				Slug: "Dex",
			},
			wantErr: "slugified form of env.Slug \"dex\" didn't match provided slug \"Dex\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			envManger := &EnvironmentManager{}
			err := envManger.Validate([]Environment{test.env}, nil)
			if test.wantErr == "" {
				req.NoError(err)
			} else {
				req.NotNil(err)
				req.Equal(test.wantErr, err.Error())
			}
		})
	}
}
