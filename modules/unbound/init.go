package unbound

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"

	"github.com/netdata/go.d.plugin/modules/unbound/config"
	"github.com/netdata/go.d.plugin/pkg/web"
)

func (u *Unbound) initConfig() (enabled bool) {
	if u.ConfPath == "" {
		u.Info("'conf_path' not set, skipping parameters auto detection")
		return true
	}

	u.Infof("reading '%s'", u.ConfPath)
	cfg, err := config.Parse(u.ConfPath)
	if err != nil {
		u.Warningf("%v, skipping parameters auto detection", err)
		return true
	}

	if cfg.Empty() {
		u.Debug("empty configuration")
		return true
	}

	if enabled, ok := cfg.ControlEnabled(); ok && !enabled {
		u.Info("remote control is disabled in the configuration file")
		return false
	}

	u.applyConfig(cfg)
	return true
}

func (u *Unbound) applyConfig(cfg *config.UnboundConfig) {
	u.Infof("applying configuration: %s", cfg)
	if cumulative, ok := cfg.Cumulative(); ok && cumulative != u.Cumulative {
		u.Debugf("changing 'cumulative_stats': %v => %v", u.Cumulative, cumulative)
		u.Cumulative = cumulative
	}
	if useCert, ok := cfg.ControlUseCert(); ok && useCert != u.UseTLS {
		u.Debugf("changing 'use_tls': %v => %v", u.UseTLS, useCert)
		u.UseTLS = useCert
	}
	if keyFile, ok := cfg.ControlKeyFile(); ok && keyFile != u.TLSKey {
		u.Debugf("changing 'tls_key': '%s' => '%s'", u.TLSKey, keyFile)
		u.TLSKey = keyFile
	}
	if certFile, ok := cfg.ControlCertFile(); ok && certFile != u.TLSCert {
		u.Debugf("changing 'tls_cert': '%s' => '%s'", u.TLSCert, certFile)
		u.TLSCert = certFile
	}
	if iface, ok := cfg.ControlInterface(); ok && adjustControlInterface(iface) != u.Address {
		address := adjustControlInterface(iface)
		u.Debugf("changing 'address': '%s' => '%s'", u.Address, address)
		u.Address = address
	}
	if port, ok := cfg.ControlPort(); ok && !isUnixSocket(u.Address) {
		if host, curPort, err := net.SplitHostPort(u.Address); err == nil && curPort != port {
			address := net.JoinHostPort(host, port)
			u.Debugf("changing 'address': '%s' => '%s'", u.Address, address)
			u.Address = address
		}
	}
}

func (u *Unbound) initClient() (err error) {
	var tlsCfg *tls.Config
	useTLS := !isUnixSocket(u.Address) && u.UseTLS

	if useTLS && (u.TLSCert == "" || u.TLSKey == "") {
		return errors.New("'tls_cert' or 'tls_key' is missing")
	}

	if useTLS {
		if tlsCfg, err = web.NewTLSConfig(u.ClientTLSConfig); err != nil {
			return err
		}
	}

	u.client = newClient(clientConfig{
		address: u.Address,
		timeout: u.Timeout.Duration,
		useTLS:  useTLS,
		tlsConf: tlsCfg,
	})
	return nil
}

func adjustControlInterface(value string) string {
	if isUnixSocket(value) {
		return value
	}
	if value == "0.0.0.0" {
		value = "127.0.0.1"
	}
	return net.JoinHostPort(value, "8953")
}

func isUnixSocket(address string) bool {
	return strings.HasPrefix(address, "/")
}
