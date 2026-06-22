package models

// GrafanaWebhook represents the JSON payload sent by Grafana alerting.
type GrafanaWebhook struct {
	Status            string            `json:"status"`
	State             string            `json:"state"`
	Title             string            `json:"title"`
	Message           string            `json:"message"`
	ExternalURL       string            `json:"externalURL"`
	Receiver          string            `json:"receiver"`
	OrgID             int               `json:"orgId"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Version           string            `json:"version"`
	AppVersion        string            `json:"appVersion"`
	GroupKey          string            `json:"groupKey"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert represents a single firing or resolved alert instance.
type Alert struct {
	Status       string             `json:"status"`
	Labels       map[string]string  `json:"labels"`
	Annotations  map[string]string  `json:"annotations"`
	StartsAt     string             `json:"startsAt"`
	EndsAt       string             `json:"endsAt"`
	GeneratorURL string             `json:"generatorURL"`
	SilenceURL   string             `json:"silenceURL"`
	PanelURL     string             `json:"panelURL"`
	DashboardURL string             `json:"dashboardURL"`
	RuleUID      string             `json:"ruleUID"`
	Fingerprint  string             `json:"fingerprint"`
	ValueString  string             `json:"valueString"`
	Values       map[string]float64 `json:"values"`
	OrgID        int                `json:"orgId"`
}
