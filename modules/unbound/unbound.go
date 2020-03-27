package unbound

import (
	"time"

	"github.com/netdata/go.d.plugin/pkg/web"

	"github.com/netdata/go-orchestrator/module"
)

func init() {
	creator := module.Creator{
		Create: func() module.Module { return New() },
	}

	module.Register("unbound", creator)
}

func New() *Unbound {
	config := Config{
		Address:    "127.0.0.1:8953",
		ConfPath:   "/etc/unbound/unbound.conf",
		Timeout:    web.Duration{Duration: time.Second},
		Cumulative: false,
		UseTLS:     true,
		ClientTLSConfig: web.ClientTLSConfig{
			TLSCert:            "/etc/unbound/unbound_control.pem",
			TLSKey:             "/etc/unbound/unbound_control.key",
			InsecureSkipVerify: true,
		},
	}

	return &Unbound{
		Config:   config,
		curCache: newCollectCache(),
		cache:    newCollectCache(),
	}
}

type unboundClient interface {
	send(command string) ([]string, error)
}

type (
	Config struct {
		Address             string       `yaml:"address"`
		ConfPath            string       `yaml:"conf_path"`
		Timeout             web.Duration `yaml:"timeout"`
		Cumulative          bool         `yaml:"cumulative_stats"`
		UseTLS              bool         `yaml:"use_tls"`
		web.ClientTLSConfig `yaml:",inline"`
	}
	Unbound struct {
		module.Base
		Config `yaml:",inline"`

		client   unboundClient
		cache    collectCache
		curCache collectCache

		prevCacheMiss    float64 // needed for cumulative mode
		extChartsCreated bool

		charts *module.Charts
	}
)

func (Unbound) Cleanup() {}

func (u *Unbound) Init() bool {
	if enabled := u.initConfig(); !enabled {
		return false
	}

	if err := u.initClient(); err != nil {
		u.Errorf("creating client: %v", err)
		return false
	}

	u.charts = charts(u.Cumulative)

	u.Debugf("using address: %s, cumulative: %v, use_tls: %v, timeout: %s", u.Address, u.Cumulative, u.UseTLS, u.Timeout)
	if u.UseTLS {
		u.Debugf("using tls_skip_verify: %v, tls_key: %s, tls_cert: %s", u.InsecureSkipVerify, u.TLSKey, u.TLSCert)
	}
	return true
}

func (u *Unbound) Check() bool {
	return len(u.Collect()) > 0
}

func (u Unbound) Charts() *module.Charts {
	return u.charts
}

func (u *Unbound) Collect() map[string]int64 {
	mx, err := u.collect()
	if err != nil {
		u.Error(err)
	}

	if len(mx) == 0 {
		return nil
	}
	return mx
}
