import { useDashboard } from "modules/dashboard/useDashboard";
import { useFeatureVisibility } from "modules/dashboard/useFeatureVisibility";
import { useDeploymentSettings } from "modules/management/DeploymentSettingsProvider";
import type { FC } from "react";
import { pageTitle } from "utils/page";
import { ObservabilitySettingsPageView } from "./ObservabilitySettingsPageView";

const ObservabilitySettingsPage: FC = () => {
	const { deploymentConfig } = useDeploymentSettings();
	const { entitlements } = useDashboard();
	const { multiple_organizations: hasPremiumLicense } = useFeatureVisibility();

	return (
		<>
			<title>{pageTitle("Observability Settings")}</title>

			<ObservabilitySettingsPageView
				options={deploymentConfig.options}
				featureAuditLogEnabled={entitlements.features.audit_log.enabled}
				isPremium={hasPremiumLicense}
			/>
		</>
	);
};

export default ObservabilitySettingsPage;
