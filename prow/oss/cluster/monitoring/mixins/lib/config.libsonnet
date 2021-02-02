local util = import 'config_util.libsonnet';

//
// Edit configuration in this object.
//
local config = {
  local comps = util.consts.components,

  // Instance specifics
  instance: {
    name: "OSS Prow",
    botName: "google-oss-robot",
    url: "https://oss-prow.knative.dev",
    monitoringURL: "https://oss-prow-monitoring.knative.dev",
  },

  // SLO compliance tracking config
  slo: {
    components: [
      comps.deck,
      comps.hook,
      comps.plank,
      comps.sinker,
      comps.tide,
      comps.monitoring,
      ],
    },

    // Tide pools that are important enough to have their own graphs on the dashboard.
    tideDashboardExplicitPools: [],

    // Additional scraping endpoints
    probeTargets: [
    # ATTENTION: Keep this in sync with the list in ../../additional-scrape-configs_secret.yaml
      // {url: 'https://oss-prow.knative.dev/monitoring', labels: {slo: comps.monitoring}},
      {url: 'https://oss-prow.knative.dev', labels: {slo: comps.deck}},
    ],

    // Boskos endpoints to be monitored
    boskosResourcetypes: [],

    // How long we go during work hours without seeing a webhook before alerting.
    webhookMissingAlertInterval: '60m',

    // How many days prow hasn't been bumped.
  prowImageStaleByDays: {daysStale: 14, eventDuration: '24h'},
};

// Generate the real config by adding in constant fields and defaulting where needed.
{
  _config+:: util.defaultConfig(config),
  _util+:: util,
}
