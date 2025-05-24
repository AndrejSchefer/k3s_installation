package nfs

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed nfs_export.sh.tmpl
var tmplFS embed.FS

func BuildScript(vars map[string]string) (string, error) {
	tmplBytes, err := tmplFS.ReadFile("nfs_export.sh.tmpl")
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("nfs").Parse(string(tmplBytes))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
