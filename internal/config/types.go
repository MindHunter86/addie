package config

// YAML config parsing
type (
	ExternalSource struct {
		Balancer *ExternalSourceBalancer `yaml:",omitempty,inline"`
	}
	ExternalSourceBalancer struct {
		Config  map[string]any              `yaml:",omitempty,inline"`
		Routing map[string]*BalancerRouting `yaml:",omitempty,inline"`
		Regions map[string]*BalancerRegion  `yaml:",omitempty,inline"`
	}
	BalancerRouting struct {
		Countries []string `yaml:",omitempty"`

		Primary   []string `yaml:",omitempty"`
		Secondary []string `yaml:",omitempty"`
		Backup    []string `yaml:",omitempty"`
	}
	BalancerRegion struct {
		Dynamic bool `yaml:",omitempty"`

		Orgin    *RegionOrigin     `yaml:",omitempty,inline"`
		Upstream []*RegionUpstream `yaml:",omitempty,inline"`
	}
	RegionOrigin struct {
		Region string `yaml:",omitempty"`
		Server string `yaml:",omitempty"`
	}
	RegionUpstream struct {
		Server    string `yaml:",omitempty"`
		Bandwidth uint   `yaml:",omitempty"`
	}
)

// urfave generic flags compatibility
type BalancerRoutingMap map[string]*BalancerRouting
type BalancerRegionMap map[string]*BalancerRegion
