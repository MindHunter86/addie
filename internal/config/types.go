package config

// YAML config parsing
type (
	ExternalSource struct {
		Balancer *ExternalSourceBalancer `yaml:",omitempty"`
	}
	ExternalSourceBalancer struct {
		Config  map[string]any              `yaml:",omitempty"`
		Routing map[string]*BalancerRouting `yaml:",omitempty"`
		Regions map[string]*BalancerRegion  `yaml:",omitempty"`
	}
	BalancerRouting struct {
		Countries []string `yaml:",omitempty"`

		Primary   []string `yaml:",omitempty"`
		Secondary []string `yaml:",omitempty"`
		Backup    []string `yaml:",omitempty"`
	}
	BalancerRegion struct {
		Dynamic bool `yaml:",omitempty"`

		Origin   *RegionOrigin     `yaml:",omitempty"`
		Upstream []*RegionUpstream `yaml:",omitempty"`
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

const epochMultiply = 5
