package result

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("summary-poster", func() {

	It("should render summary with 1 and 2 results for the same line", func() {
		run := &testrunner.Run{
			Info: nil,
			Testrun: &tmv1beta1.Testrun{
				Status: tmv1beta1.TestrunStatus{
					Phase: tmv1beta1.RunPhaseSuccess,
				},
			},
			Metadata: &metadata.Metadata{
				CloudProvider: "foo",
			},
			Error:      nil,
			Rerenderer: nil,
		}
		var runs testrunner.RunList
		runs = append(runs, run)
		items := parseTestrunsToTableItems(runs)
		Expect(items).To(HaveLen(1))
		slack, err := util.RenderTableForSlack(nil, items)
		Expect(err).ToNot(HaveOccurred())
		Expect(slack).To(ContainSubstring(string(util.StatusSymbolSuccess)))
		Expect(slack).ToNot(ContainSubstring(string(util.StatusSymbolNA)))

		run = &testrunner.Run{
			Info: nil,
			Testrun: &tmv1beta1.Testrun{
				Status: tmv1beta1.TestrunStatus{
					Phase: tmv1beta1.RunPhaseFailed,
				},
			},
			Metadata: &metadata.Metadata{
				CloudProvider: "foo",
			},
			Error:      nil,
			Rerenderer: nil,
		}
		runs = append(runs, run)
		items = parseTestrunsToTableItems(runs)
		Expect(items).To(HaveLen(2))
		slack, err = util.RenderTableForSlack(nil, items)
		Expect(err).ToNot(HaveOccurred())
		Expect(slack).To(ContainSubstring(string(util.StatusSymbolSuccess) + util.SymbolOffset + string(util.StatusSymbolFailure)))
		Expect(slack).ToNot(ContainSubstring(string(util.StatusSymbolNA)))
	})
})
