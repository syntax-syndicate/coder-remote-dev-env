package prebuilds

import (
	"context"
	"time"

	"cdr.dev/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/prebuilds"
)

var (
	CreatedPrebuildsDesc   = prometheus.NewDesc("coderd_prebuilds_created", "The number of prebuilds created.", []string{"template_name", "preset_name"}, nil)
	FailedPrebuildsDesc    = prometheus.NewDesc("coderd_prebuilds_failed", "The number of prebuilds that failed.", []string{"template_name", "preset_name"}, nil)
	AssignedPrebuildsDesc  = prometheus.NewDesc("coderd_prebuilds_assigned", "The number of prebuilds that were assigned to a runner.", []string{"template_name", "preset_name"}, nil)
	UsedPresetsDesc        = prometheus.NewDesc("coderd_prebuilds_used_presets", "The number of times a preset was used to build a prebuild.", []string{"template_name", "preset_name"}, nil)
	ExhaustedPrebuildsDesc = prometheus.NewDesc("coderd_prebuilds_exhausted", "The number of prebuilds that were exhausted.", []string{"template_name", "preset_name"}, nil)
	DesiredPrebuildsDesc   = prometheus.NewDesc("coderd_prebuilds_desired", "The number of desired prebuilds.", []string{"template_name", "preset_name"}, nil)
	ActualPrebuildsDesc    = prometheus.NewDesc("coderd_prebuilds_actual", "The number of actual prebuilds.", []string{"template_name", "preset_name"}, nil)
	EligiblePrebuildsDesc  = prometheus.NewDesc("coderd_prebuilds_eligible", "The number of eligible prebuilds.", []string{"template_name", "preset_name"}, nil)
)

type MetricsCollector struct {
	database   database.Store
	logger     slog.Logger
	reconciler prebuilds.Reconciler
}

var _ prometheus.Collector = new(MetricsCollector)

func NewMetricsCollector(db database.Store, logger slog.Logger, reconciler prebuilds.Reconciler) *MetricsCollector {
	return &MetricsCollector{
		database:   db,
		logger:     logger.Named("prebuilds_metrics_collector"),
		reconciler: reconciler,
	}
}

func (*MetricsCollector) Describe(descCh chan<- *prometheus.Desc) {
	descCh <- CreatedPrebuildsDesc
	descCh <- FailedPrebuildsDesc
	descCh <- AssignedPrebuildsDesc
	descCh <- UsedPresetsDesc
	descCh <- ExhaustedPrebuildsDesc
	descCh <- DesiredPrebuildsDesc
	descCh <- ActualPrebuildsDesc
	descCh <- EligiblePrebuildsDesc
}

func (mc *MetricsCollector) Collect(metricsCh chan<- prometheus.Metric) {
	// TODO (sasswart): get a proper actor in here, to deescalate from system
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// nolint:gocritic // just until we get back to this
	prebuildCounters, err := mc.database.GetPrebuildMetrics(dbauthz.AsSystemRestricted(ctx))
	if err != nil {
		mc.logger.Error(ctx, "failed to get prebuild metrics", slog.Error(err))
		return
	}

	for _, metric := range prebuildCounters {
		metricsCh <- prometheus.MustNewConstMetric(CreatedPrebuildsDesc, prometheus.CounterValue, float64(metric.CreatedCount), metric.TemplateName, metric.PresetName)
		metricsCh <- prometheus.MustNewConstMetric(FailedPrebuildsDesc, prometheus.CounterValue, float64(metric.FailedCount), metric.TemplateName, metric.PresetName)
		metricsCh <- prometheus.MustNewConstMetric(AssignedPrebuildsDesc, prometheus.CounterValue, float64(metric.ClaimedCount), metric.TemplateName, metric.PresetName)
	}

	state, err := mc.reconciler.SnapshotState(dbauthz.AsSystemRestricted(ctx), mc.database)
	if err != nil {
		mc.logger.Error(ctx, "failed to get latest prebuild state", slog.Error(err))
		return
	}

	for _, preset := range state.Presets {
		if !preset.UsingActiveVersion {
			continue
		}

		presetState, err := state.FilterByPreset(preset.PresetID)
		if err != nil {
			mc.logger.Error(ctx, "failed to filter by preset", slog.Error(err))
			continue
		}
		actions, err := mc.reconciler.DetermineActions(ctx, *presetState)
		if err != nil {
			mc.logger.Error(ctx, "failed to determine actions", slog.Error(err))
			continue
		}

		metricsCh <- prometheus.MustNewConstMetric(DesiredPrebuildsDesc, prometheus.GaugeValue, float64(actions.Desired), preset.TemplateName, preset.Name)
		metricsCh <- prometheus.MustNewConstMetric(ActualPrebuildsDesc, prometheus.GaugeValue, float64(actions.Actual), preset.TemplateName, preset.Name)
		metricsCh <- prometheus.MustNewConstMetric(EligiblePrebuildsDesc, prometheus.GaugeValue, float64(actions.Eligible), preset.TemplateName, preset.Name)
	}
}
