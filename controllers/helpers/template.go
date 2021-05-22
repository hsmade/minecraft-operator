package helpers

import (
	"bytes"
	"github.com/pkg/errors"
	"text/template"
)

func RenderTemplate(templateString string, data interface{}) (error, string) {
	t, err := template.New("template").Parse(templateString)
	if err != nil {
		return errors.Wrap(err, "loading template"), ""
	}

	result := new(bytes.Buffer)
	err = t.Execute(result, data)
	if err != nil {
		return errors.Wrap(err, "parsing template"), ""
	}
	return nil, result.String()
}
