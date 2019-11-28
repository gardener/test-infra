package template

import (
	"fmt"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrun_renderer"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/strvals"
)

// templateRenderer is the internal template templateRenderer
type templateRenderer struct {
	log      logr.Logger
	renderer chartrenderer.Interface

	setValues  string
	fileValues []string
}

func newRenderer(log logr.Logger, setValues string, fileValues []string) (*templateRenderer, error) {
	chartRenderer := chartrenderer.New(engine.New(), &chartutil.Capabilities{})
	return &templateRenderer{
		log:        log,
		renderer:   chartRenderer,
		setValues:  setValues,
		fileValues: fileValues,
	}, nil
}

func (r *templateRenderer) RenderChart(parameters *internalParameters, chartPath string, values map[string]interface{}, metadata *testrunner.Metadata, info interface{}) (testrunner.RunList, error) {
	if chartPath == "" {
		return make(testrunner.RunList, 0), nil
	}
	var err error

	values, err = r.determineValues(values)
	if err != nil {
		return nil, err
	}
	r.log.V(3).Info(fmt.Sprintf("Values: \n%s \n", util.PrettyPrintStruct(values)))

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
			Info:     info,
			Testrun:  tr,
			Metadata: meta,
		}
	}
	return renderedTestruns, nil
}

// determineValues fetches values from all specified files and set values.
// The values are merged with the defaultValues whereas file values overwrite default values and set values overwrite file values.
func (r *templateRenderer) determineValues(defaultValues map[string]interface{}) (map[string]interface{}, error) {
	newFileValues, err := readFileValues(r.fileValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read values from file")
	}
	defaultValues = utils.MergeMaps(defaultValues, newFileValues)
	newSetValues, err := strvals.ParseString(r.setValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse set values")
	}
	defaultValues = utils.MergeMaps(defaultValues, newSetValues)

	return defaultValues, nil
}

func ParseTestrunsFromChart(log logr.Logger, chart *chartrenderer.RenderedChart) []*v1beta1.Testrun {
	testruns := make([]*v1beta1.Testrun, 0)
	for _, file := range chart.Files() {
		tr, err := util.ParseTestrun([]byte(file))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file: %s", err.Error()))
			continue
		}
		testruns = append(testruns, &tr)
	}
	return testruns
}
