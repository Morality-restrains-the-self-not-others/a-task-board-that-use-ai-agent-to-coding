import React, { useEffect } from 'react';
import { locationService } from '@grafana/runtime';
import { LoadingPlaceholder } from '@grafana/ui';
import pluginJson from '../plugin.json';
import { testIds } from '../components/testIds';

/** App root: redirect to Daydaymoney 控制台 (plugin config page). Not shown in nav. */
function HomeRedirect() {
  useEffect(() => {
    locationService.push(`/plugins/${pluginJson.id}`);
  }, []);

  return (
    <div data-testid={testIds.homeRedirect.container}>
      <LoadingPlaceholder text="正在打开 Daydaymoney 控制台…" />
    </div>
  );
}

export default HomeRedirect;
