package remoteaccess

import "testing"

func TestParseConfigurationName(t *testing.T) {
	cases := []struct {
		name string
		want ConfigurationName
	}{
		{"http:Node-RED", ConfigurationName{Scheme: "http", Label: "Node-RED"}},
		{"https:Router", ConfigurationName{Scheme: "https", Label: "Router"}},
		{"http+mux:Grafana", ConfigurationName{Scheme: "http", Multiplex: true, Label: "Grafana"}},
		{"https+mux:a:b", ConfigurationName{Scheme: "https", Multiplex: true, Label: "a:b"}},
		{"http+other+mux:x", ConfigurationName{Scheme: "http", Multiplex: true, Label: "x"}},
		{"http+other:x", ConfigurationName{Scheme: "http", Label: "x"}},
	}
	for _, c := range cases {
		got, ok := ParseConfigurationName(c.name)
		if !ok || got != c.want {
			t.Errorf("ParseConfigurationName(%q) = %+v, %v, want %+v", c.name, got, ok, c.want)
		}
	}
}

func TestParseConfigurationNameIgnoresOtherNames(t *testing.T) {
	for _, name := range []string{"ssh", "passthrough", "httpx:a", "mux:http", "HTTP:x", "http+MUX:x", ""} {
		if got, ok := ParseConfigurationName(name); ok {
			t.Errorf("ParseConfigurationName(%q) = %+v, want no match", name, got)
		}
	}
}
