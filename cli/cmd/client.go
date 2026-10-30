package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"claimctl-cli/pkg/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newClient(cmd *cobra.Command) (*api.Client, error) {
	token := viper.GetString("token")
	explicitToken := cmd.Flag("token") != nil && cmd.Flag("token").Changed
	if path := viper.GetString("token-file"); path != "" && !explicitToken {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return nil, fmt.Errorf("token file requires owner-only permissions")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		token = strings.TrimSpace(string(data))
	}
	client, err := api.NewClient(viper.GetString("url"), token, viper.GetBool("netrc"))
	if err != nil {
		return nil, err
	}
	u, _ := neturl.Parse(client.BaseURL)
	ip := net.ParseIP(u.Hostname())
	if u.Scheme == "http" && u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) && !viper.GetBool("allow-insecure-http") {
		return nil, fmt.Errorf("HTTPS required for remote servers; use allow-insecure-http only on trusted networks")
	}
	timeout := viper.GetDuration("request-timeout")
	if timeout <= 0 {
		return nil, fmt.Errorf("request-timeout must be positive")
	}
	client.HttpClient.Timeout = timeout
	client.Context = cmd.Context()
	if path := viper.GetString("ca-cert"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("CA file contains no certificates")
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
		client.HttpClient.Transport = transport
	}
	return client, nil
}
func outputJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.String("token-file", "", "Read API token from an owner-only file")
	flags.String("ca-cert", "", "Additional trusted CA PEM file")
	flags.Duration("request-timeout", 30*time.Second, "HTTP request timeout")
	flags.Bool("allow-insecure-http", false, "Permit remote plaintext HTTP")
	for _, key := range []string{"token-file", "ca-cert", "request-timeout", "allow-insecure-http"} {
		_ = viper.BindPFlag(key, flags.Lookup(key))
	}
}
