package test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/helm"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var (
	tplPrometheusRule = []string{"templates/prometheus/prometheusrule.yaml"}
)

func Test_PrometheusRule_GivenDisabled_WhenAdditionalRulesDefined_ThenRenderNoTemplate(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"metrics.prometheusRule.enabled": "false",
		},
		ValuesFiles: []string{"testdata/custom_rules"},
	}

	_, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, tplPrometheusRule)
	assert.Error(t, err)
}

func Test_PrometheusRule_GivenEnabled_WhenNamespaceDefined_ThenRenderNewNamespace(t *testing.T) {
	expectedNamespace := "alternative"
	options := &helm.Options{
		SetValues: map[string]string{
			"metrics.prometheusRule.enabled":   "true",
			"metrics.prometheusRule.namespace": expectedNamespace,
		},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
	rule := monitoringv1.PrometheusRule{}
	helm.UnmarshalK8SYaml(t, output, &rule)

	assert.Equal(t, expectedNamespace, rule.Namespace)
}

func Test_PrometheusRule_GivenEnabled_WhenAdditionalLabelsDefined_ThenRenderMoreLabels(t *testing.T) {
	expectedLabelKey := "my-custom-label"
	expectedLabelValue := "my-value"
	options := &helm.Options{
		ValuesFiles: []string{"testdata/labels.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
	rule := monitoringv1.PrometheusRule{}
	helm.UnmarshalK8SYaml(t, output, &rule)

	assert.Equal(t, expectedLabelValue, rule.Labels[expectedLabelKey])
}

func Test_PrometheusRule_GivenEnabled_WhenCreateDefaultRulesEnabled_ThenRenderDefaultAlerts(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"metrics.prometheusRule.enabled": "true",
		},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
	rule := monitoringv1.PrometheusRule{}
	helm.UnmarshalK8SYaml(t, output, &rule)

	assert.NotEmpty(t, rule.Spec.Groups[0].Rules)
	assert.GreaterOrEqual(t, len(rule.Spec.Groups[0].Rules), 4)
}

var legacyRuleSubjects = map[string]struct {
	legacyRulesEnabled  bool
	expectRuleToContain string
}{
	"WhenLegacyRulesDisabled_ThenRenderNormalRule": {
		true, "by(job,",
	},
	"WhenLegacyRulesEnabled_ThenRenderLegacyRule": {
		false, "by(job_name,",
	},
}

func Test_PrometheusRule_GivenEnabled_LegacyRules(t *testing.T) {
	for descr, tC := range legacyRuleSubjects {
		t.Run(descr, func(t *testing.T) {
			options := &helm.Options{
				SetValues: map[string]string{
					"metrics.prometheusRule.enabled":     "true",
					"metrics.prometheusRule.legacyRules": strconv.FormatBool(tC.legacyRulesEnabled),
				},
			}

			output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
			rule := monitoringv1.PrometheusRule{}
			helm.UnmarshalK8SYaml(t, output, &rule)

			findFailedRule := func(rules []monitoringv1.Rule) *monitoringv1.Rule {
				for _, rule := range rules {
					if strings.HasSuffix(rule.Alert, "Failed") {
						return &rule
					}
				}
				return nil
			}

			failedRule := findFailedRule(rule.Spec.Groups[0].Rules)
			assert.NotNil(t, failedRule)
			assert.Contains(t, failedRule.Expr.String(), tC.expectRuleToContain)
		})
	}
}

func Test_PrometheusRule_GivenEnabled_WhenCreateDefaultRulesDisabled_ThenRenderNoTemplate(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"metrics.prometheusRule.enabled":            "true",
			"metrics.prometheusRule.createDefaultRules": "false",
		},
	}

	_, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, tplPrometheusRule)
	assert.Error(t, err)
}

func Test_PrometheusRule_GivenEnabled_WhenCreateDefaultRulesDisabledAndAdditionalRulesGiven_ThenRenderCustomRules(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"metrics.prometheusRule.createDefaultRules": "false",
		},
		ValuesFiles: []string{"testdata/custom_rules.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
	rule := monitoringv1.PrometheusRule{}
	helm.UnmarshalK8SYaml(t, output, &rule)

	assert.Equal(t, 1, len(rule.Spec.Groups[0].Rules))
	assert.Equal(t, "MyCustomRule", rule.Spec.Groups[0].Rules[0].Alert)
}

func Test_PrometheusRule_GivenEnabled_WhenCreateDefaultRulesEnabledAndAdditionalRulesGiven_ThenRenderDefaultAndCustomRules(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"testdata/custom_rules.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplPrometheusRule)
	rule := monitoringv1.PrometheusRule{}
	helm.UnmarshalK8SYaml(t, output, &rule)

	amount := len(rule.Spec.Groups[0].Rules)
	assert.GreaterOrEqual(t, amount, 2)
	assert.Equal(t, "MyCustomRule", rule.Spec.Groups[0].Rules[amount-1].Alert)
}
