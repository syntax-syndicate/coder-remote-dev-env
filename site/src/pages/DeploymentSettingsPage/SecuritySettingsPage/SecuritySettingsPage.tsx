import { useDashboard } from "modules/dashboard/useDashboard";
import { useDeploymentSettings } from "modules/management/DeploymentSettingsProvider";
import type { FC } from "react";
import { pageTitle } from "utils/page";
import { SecuritySettingsPageView } from "./SecuritySettingsPageView";

const SecuritySettingsPage: FC = () => {
	const { deploymentConfig } = useDeploymentSettings();
	const { entitlements } = useDashboard();

	return (
		<>
			<title>{pageTitle("Security Settings")}</title>

			<SecuritySettingsPageView
				options={deploymentConfig.options}
				featureBrowserOnlyEnabled={entitlements.features.browser_only.enabled}
			/>
		</>
	);
};

export default SecuritySettingsPage;
