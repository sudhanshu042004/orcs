package site

import "testing"

func TestDeploymentIdFromHost(t *testing.T) {
	yes := map[string]int64{
		"20.localhost:3000":     20,
		"20.localhost":          20,
		"7.orcs.dev":            7,
		"123.sites.example.com": 123,
	}
	for host, want := range yes {
		got, ok := deploymentIdFromHost(host)
		if !ok || got != want {
			t.Errorf("%q -> %d (ok=%v), want %d", host, got, ok, want)
		}
	}
	no := []string{
		"localhost:3000", "localhost", "orcs.dev", "api.orcs.dev",
		"", "0.localhost", "-1.localhost", "20a.localhost", ".localhost",
		"[::1]:3000", "127.0.0.1:3000",
	}
	for _, host := range no {
		if id, ok := deploymentIdFromHost(host); ok {
			t.Errorf("%q should not be a site host, got %d", host, id)
		}
	}
}

func TestObjectKeyStaysInsidePrefix(t *testing.T) {
	cases := map[string]string{
		"/":                    "deployments/20/index.html",
		"/index.html":          "deployments/20/index.html",
		"/assets/index-ab.js":  "deployments/20/assets/index-ab.js",
		"/favicon.ico":         "deployments/20/favicon.ico",
		"/about":               "deployments/20/about",
		"/nested/":             "deployments/20/nested/index.html",
		"/../../etc/passwd":    "deployments/20/etc/passwd",
		"/assets/../../secret": "deployments/20/secret",
		"//evil":               "deployments/20/evil",
	}
	for in, want := range cases {
		if got := objectKey(20, in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}
