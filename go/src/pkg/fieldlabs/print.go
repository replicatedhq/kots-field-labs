package fieldlabs

import (
	"github.com/replicatedhq/replicated/pkg/types"
	"io"
	"text/tabwriter"
	"text/template"
)

const (
	minWidth = 0
	tabWidth = 8
	padding  = 4
	padChar  = ' '
)

func tabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, minWidth, tabWidth, padding, padChar, tabwriter.TabIndent)
}

var appsTmpl = template.Must(template.New("apps").Parse(appsTmplSrc))
var appsTmplSrc = `ID	NAME	SLUG	SCHEDULER
{{ range . -}}
{{ .ID }}	{{ .Name }}	{{ .Slug}}	{{ .Scheduler }}
{{ end }}`

func (e *EnvironmentManager) PrintApps(apps []types.App) error {
	if len(apps) == 0 {
		return nil
	}

	w := tabWriter(e.Writer)

	if err := appsTmpl.Execute(w, apps); err != nil {
		return err
	}

	return w.Flush()
}

var channelAttrsTmpl = template.Must(template.New("ChannelAttributes").Parse(channelAttrsTmplSrc))
var channelAttrsTmplSrc = `ID:	{{ .ID }}
NAME:	{{ .Name }}
DESCRIPTION:	{{ .Description }}
RELEASE:	{{ if ge .ReleaseSequence 1 }}{{ .ReleaseSequence }}{{else}}	{{end}}
VERSION:	{{ .ReleaseLabel }}{{ with .InstallCommands }}
EXISTING:

{{ .Existing }}

EMBEDDED:

{{ .Embedded }}

AIRGAP:

{{ .Airgap }}
{{end}}
`

func (e *EnvironmentManager) PrintChannel(appChan types.Channel) error {

	w := tabWriter(e.Writer)
	if err := channelAttrsTmpl.Execute(w, appChan); err != nil {
		return err
	}
	return w.Flush()
}
