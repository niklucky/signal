package models

// Host defines a scheduled HTTP health check.
type Host struct {
	Name           string            `yaml:"name"`
	Method         string            `yaml:"method"`
	URL            string            `yaml:"url"`
	Headers        map[string]string `yaml:"headers"`
	Body           string            `yaml:"body"`
	Timeout        int               `yaml:"timeout"`
	Interval       int               `yaml:"interval"`
	ResendInterval int               `yaml:"resend_interval"`
}

// HostsFile is the top-level structure of hosts.yml.
type HostsFile struct {
	Hosts []Host `yaml:"hosts"`
}
