package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

var (
	ConfigFile    string
	DefaultConfig = `[registry]\n
proxy_host = ""\n
[release]\n
proxy_host = ""\n
path_prefix = ""\n
terraform_version = "1.1.7"\n
[server]\n
address = ""\n
is_private = false\n
cert_file = ""\n
key_file = ""\n
use_tls = false`
)

// WebReverseProxyConfiguration is a coniguration for the ReverseProxy
type WebReverseProxyConfiguration struct {
	RegistryProxyHost string
	ReleaseProxyHost  string
	ReleasePathPrefix string
	HttpAddress       string
	TerraformVersion  string
	IsPrivate         bool
	UseTLS            TLS
}

type TLS struct {
	CertFile string
	KeyFile  string
	Use      bool
}

func loadIni() *WebReverseProxyConfiguration {
	config, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}
	return &WebReverseProxyConfiguration{
		RegistryProxyHost: config.Section("registry").Key("proxy_host").String(),
		ReleaseProxyHost:  config.Section("release").Key("proxy_host").String(),
		ReleasePathPrefix: config.Section("release").Key("path_prefix").String(),
		HttpAddress:       config.Section("server").Key("address").String(),
		TerraformVersion:  config.Section("server").Key("terraform_version").String(),
		IsPrivate:         config.Section("server").Key("is_private").MustBool(false),
		UseTLS: TLS{
			CertFile: config.Section("server").Key("cert_file").String(),
			KeyFile:  config.Section("server").Key("key_file").String(),
			Use:      config.Section("server").Key("use_tls").MustBool(false),
		},
	}
}

// This replaces all occurrences of http://releases.hashicorp.com with
// config.ReleaseProxyHost in the response body
func (config *WebReverseProxyConfiguration) rewriteBody(resp *http.Response) (err error) {
	// Check that the server actually sent compressed data
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = -1
		resp.Uncompressed = true
		defer func(reader io.ReadCloser) {
			err := reader.Close()
			if err != nil {
				log.Printf("Failed to close gzip reader: %v", err)
			}
		}(reader)
	default:
		reader = resp.Body
	}

	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if err = resp.Body.Close(); err != nil {
		return err
	}

	replacement := fmt.Sprintf("https://%s%s", config.ReleaseProxyHost, config.ReleasePathPrefix)

	b = bytes.ReplaceAll(b, []byte("https://releases.hashicorp.com"), []byte(replacement)) // releases
	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	resp.ContentLength = int64(len(b))
	resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
	return nil
}

func (config *WebReverseProxyConfiguration) NewWebReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if req.Host == config.RegistryProxyHost {
			req.URL.Scheme = "https"
			req.URL.Host = "registry.terraform.io"
			req.Host = "registry.terraform.io"
			req.Header.Set("User-Agent", fmt.Sprintf("Terraform/%s", config.TerraformVersion))
			req.Header.Set("X-Terraform-Version", config.TerraformVersion)
		} else if req.Host == config.ReleaseProxyHost {
			req.URL.Scheme = "https"
			req.URL.Host = "releases.hashicorp.com"
			req.Host = "releases.hashicorp.com"
			req.Header.Set("User-Agent", fmt.Sprintf("Terraform/%s", config.TerraformVersion))
		}
	}

	responseDirector := func(res *http.Response) error {
		if server := res.Header.Get("Server"); strings.HasPrefix(server, "terraform-registry") {
			if err := config.rewriteBody(res); err != nil {
				fmt.Println("Error rewriting body!")
				return err
			}
		}

		if location := res.Header.Get("Location"); location != "" {
			requestURI, err := url.ParseRequestURI(location)
			if err != nil {
				fmt.Println("Error!")
				return err
			}

			// Override redirect requestURI Host with ProxyHost
			requestURI.Host = config.RegistryProxyHost

			res.Header.Set("Location", requestURI.String())
			res.Header.Set("X-Reverse-Proxy", "terraform-registry-proxy")
		}
		return nil
	}

	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: responseDirector,
	}
}
