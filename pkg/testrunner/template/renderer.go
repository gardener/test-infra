package template

import (
	"fmt"
	"github.com/gardener/gardener/pkg/chartrenderer"
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
	"k8s.io/helm/pkg/strvals"
)

// templateRenderer is the internal template templateRenderer
type templateRenderer struct {
	log      logr.Logger
	renderer chartrenderer.Interface

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
	chartRenderer := chartrenderer.New(engine.New(), &chartutil.Capabilities{})
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

// RenderChart renders a helm chart of multiple testruns with values and returns a list of runs.
func (r *templateRenderer) RenderChart(parameters *internalParameters, chartPath string, valueRenderer ValueRenderer) (testrunner.RunList, error) {
	if chartPath == "" {
		return make(testrunner.RunList, 0), nil
	}

	state := &renderState{
		templateRenderer: *r,
		chartPath:        chartPath,
		values:           valueRenderer,
		parameters:       parameters,
	}

	values, metadata, info, err := valueRenderer.Render(r.defaultValues)
	if err != nil {
		return nil, err
	}
	chart, err := r.renderer.Render(chartPath, "", "", values)
	if err != nil {
		return nil, err
	}

	testruns := ParseTestrunsFromChart(r.log, chart)
	renderedTestruns := make(testrunner.RunList, len(testruns))
	for i, tr := range testruns {
		meta := metadata.DeepCopy()
		// Add all repositories defined in the component descriptor to the testrun locations.
		// This gives us all dependent repositories as well as there deployed version.
		if err := testrun_renderer.AddBOMLocationsToTestrun(tr, "default", parameters.ComponentDescriptor, true); err != nil {
			r.log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
			continue
		}

		// Add runtime annotations to the testrun
		addAnnotationsToTestrun(tr, meta.CreateAnnotations())

		renderedTestruns[i] = &testrunner.Run{
			Info:       info,
			Testrun:    tr,
			Metadata:   meta,
			Rerenderer: state,
		}
	}
	return renderedTestruns, nil
}

func (s *renderState) Rerender(tr *v1beta1.Testrun) (*testrunner.Run, error) {
	runs, err := s.RenderChart(s.parameters, s.chartPath, s.values)
	if err != nil {
		return nil, err
	}

	templateID, ok := tr.GetAnnotations()[common.TemplateIDTestrunAnnotation]
	if !ok {
		return nil, errors.Errorf("testrun %s does not have a template id", tr.GetName())
	}
	for _, run := range runs {
		if id := run.Testrun.GetAnnotations()[common.TemplateIDTestrunAnnotation]; id == templateID {
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

func ParseTestrunsFromChart(log logr.Logger, chart *chartrenderer.RenderedChart) []*v1beta1.Testrun {
	testruns := make([]*v1beta1.Testrun, 0)
	for filename, file := range chart.Files() {
		tr, err := util.ParseTestrun([]byte(file))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file: %s", err.Error()))
			continue
		}
		metav1.SetMetaDataAnnotation(&tr.ObjectMeta, common.TemplateIDTestrunAnnotation, filename)
		testruns = append(testruns, &tr)
	}
	return testruns
}
