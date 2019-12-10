package template

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testrun_renderer"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	chartapi "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/strvals"
	"k8s.io/helm/pkg/timeconv"
	"path/filepath"
)

// templateRenderer is the internal template templateRenderer
type templateRenderer struct {
	log      logr.Logger
	renderer *engine.Engine

	defaultValues map[string]interface{}
}

// renderState holds the state of a chart rendering.
// It implements the rerender interface to that the same chart with equal configuration can be retried.
type renderState struct {
	templateRenderer
	chartPath  string
	values     ValueRenderer
	parameters *internalParameters
}

func newTemplateRenderer(log logr.Logger, setValues string, fileValues []string) (*templateRenderer, error) {
	chartRenderer := engine.New()
	values, err := determineDefaultValues(setValues, fileValues)
	if err != nil {
		return nil, err
	}
	log.V(3).Info(fmt.Sprintf("Values: \n%s \n", util.PrettyPrintStruct(values)))
	return &templateRenderer{
		log:           log,
		renderer:      chartRenderer,
		defaultValues: values,
	}, nil
}

// Render renders a helm chart of multiple testruns with values and returns a list of runs.
func (r *templateRenderer) Render(parameters *internalParameters, chartPath string, valueRenderer ValueRenderer) (testrunner.RunList, error) {
	if chartPath == "" {
		return make(testrunner.RunList, 0), nil
	}

	state := &renderState{
		templateRenderer: *r,
		chartPath:        chartPath,
		values:           valueRenderer,
		parameters:       parameters,
	}

	c, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	// split all found templates into separate charts
	templates, files := splitTemplates(c.GetTemplates())
	runs := make(testrunner.RunList, 0)
	for _, tmpl := range files {
		values, metadata, info, err := valueRenderer.Render(r.defaultValues)
		if err != nil {
			return nil, err
		}
		c.Templates = append(templates, tmpl)
		files, err := r.RenderChart(c, parameters.Namespace, values)
		if err != nil {
			return nil, err
		}

		testruns := parseTestrunsFromChart(r.log, files)

		for _, tr := range testruns {
			meta := metadata.DeepCopy()
			// Add all repositories defined in the component descriptor to the testrun locations.
			// This gives us all dependent repositories as well as there deployed version.
			if err := testrun_renderer.AddBOMLocationsToTestrun(tr, "default", parameters.ComponentDescriptor, true); err != nil {
				r.log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
				continue
			}

			// Add runtime annotations to the testrun
			addAnnotationsToTestrun(tr, meta.CreateAnnotations())

			runs = append(runs, &testrunner.Run{
				Info:       info,
				Testrun:    tr,
				Metadata:   meta,
				Rerenderer: state,
			})
		}

	}

	return runs, nil
}

func (r *templateRenderer) RenderChart(chart *chartapi.Chart, namespace string, values map[string]interface{}) (map[string]string, error) {
	chartName := chart.GetMetadata().GetName()

	parsedValues, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("can't parse variables for chart %s: ,%s", chartName, err)
	}
	chartConfig := &chartapi.Config{Raw: string(parsedValues)}

	err = chartutil.ProcessRequirementsEnabled(chart, chartConfig)
	if err != nil {
		return nil, fmt.Errorf("can't process requirements for chart %s: ,%s", chartName, err)
	}
	err = chartutil.ProcessRequirementsImportValues(chart)
	if err != nil {
		return nil, fmt.Errorf("can't process requirements for import values for chart %s: ,%s", chartName, err)
	}

	revision := 1
	ts := timeconv.Now()
	options := chartutil.ReleaseOptions{
		Name:      util.RandomString(5),
		Time:      ts,
		Namespace: namespace,
		Revision:  revision,
		IsInstall: true,
	}

	valuesToRender, err := chartutil.ToRenderValuesCaps(chart, chartConfig, options, nil)
	if err != nil {
		return nil, err
	}
	return r.renderer.Render(chart, valuesToRender)
}

func (s *renderState) Rerender(tr *v1beta1.Testrun) (*testrunner.Run, error) {
	runs, err := s.Render(s.parameters, s.chartPath, s.values)
	if err != nil {
		return nil, err
	}

	templateID, ok := tr.GetAnnotations()[common.AnnotationTemplateIDTestrun]
	if !ok {
		return nil, errors.Errorf("testrun %s does not have a template id", tr.GetName())
	}
	for _, run := range runs {
		if id := run.Testrun.GetAnnotations()[common.AnnotationTemplateIDTestrun]; id == templateID {
			return run, nil
		}
	}
	return nil, errors.Errorf("unable to rerender testrun for file %s", "")
}

// determineDefaultValues fetches values from all specified files and set values.
// The values are merged whereas set values overwrite file values.
func determineDefaultValues(setValues string, fileValues []string) (map[string]interface{}, error) {
	values, err := readFileValues(fileValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read values from file")
	}
	newSetValues, err := strvals.ParseString(setValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse set values")
	}
	values = utils.MergeMaps(values, newSetValues)
	return values, nil
}

// splitTemplates splits all found templates into 2 lists of .tpl and other files
// todo: improve to check whether the template is a testrun
func splitTemplates(all []*chartapi.Template) ([]*chartapi.Template, []*chartapi.Template) {
	var (
		templates = make([]*chartapi.Template, 0)
		others    = make([]*chartapi.Template, 0)
	)
	for _, tmpl := range all {
		if filepath.Ext(tmpl.Name) == ".tpl" {
			templates = append(templates, tmpl)
		} else {
			others = append(others, tmpl)
		}
	}
	return templates, others
}

func parseTestrunsFromChart(log logr.Logger, files map[string]string) []*v1beta1.Testrun {
	testruns := make([]*v1beta1.Testrun, 0)
	for filename, file := range files {
		tr, err := util.ParseTestrun([]byte(file))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file: %s", err.Error()))
			continue
		}
		metav1.SetMetaDataAnnotation(&tr.ObjectMeta, common.AnnotationTemplateIDTestrun, filename)
		testruns = append(testruns, &tr)
	}
	return testruns
}
