package render_template

import (
	"bytes"
	"text/template"
)

// RenderLocalTemplate uses a template <tpl> given as a string and renders it. Thus, the template does not
// necessarily need to be stored as a file.
func RenderLocalTemplate(tpl string, values interface{}) ([]byte, error) {
	templateObj, err := template.
		New("tpl").
		Parse(tpl)
	if err != nil {
		return nil, err
	}
	return render(templateObj, values)
}

// render takes a text/template.Template object <temp> and an interface of <values> which are used to render the
// template. It returns the rendered result as byte slice, or an error if something went wrong.
func render(tpl *template.Template, values interface{}) ([]byte, error) {
	var result bytes.Buffer
	err := tpl.Execute(&result, values)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}
