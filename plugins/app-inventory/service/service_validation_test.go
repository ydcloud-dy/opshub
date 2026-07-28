package service

import "testing"

func TestValidateApplicationInputNormalizesTags(t *testing.T) {
	in := &ApplicationInput{
		Code: "order-api", Name: "Order API", OwnerUserID: 7, DepartmentID: 3, EnvironmentID: 2,
		Criticality: "high", Status: "active",
		RepositoryURL: "https://git.example.com/trade/order-api", Tags: `["核心链路","核心链路","订单"]`,
	}
	if err := validateApplicationInput(in); err != nil {
		t.Fatalf("valid application rejected: %v", err)
	}
	if in.Tags != `["核心链路","订单"]` {
		t.Fatalf("tags were not normalized: %s", in.Tags)
	}
}

func TestValidateApplicationInputRejectsInvalidGovernanceFields(t *testing.T) {
	tests := []ApplicationInput{
		{Code: "bad code", Name: "Application", OwnerUserID: 1, EnvironmentID: 1},
		{Code: "app", Name: "Application", EnvironmentID: 1},
		{Code: "app", Name: "Application", OwnerUserID: 1, EnvironmentID: 1, RepositoryURL: "ssh://git.example.com/repo"},
		{Code: "app", Name: "Application", OwnerUserID: 1, EnvironmentID: 1, Tags: `{"name":"tag"}`},
	}
	for i := range tests {
		if err := validateApplicationInput(&tests[i]); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestValidateAssetInputsRequireConcreteLocation(t *testing.T) {
	if err := validateDomainInput(&DomainInput{Protocol: "https", Port: 443, Path: "/"}); err == nil {
		t.Fatal("empty domain should be rejected")
	}
	if err := validateResourceInput(&ResourceInput{Kind: "Deployment", Name: "api", ClusterID: 2}); err == nil {
		t.Fatal("kubernetes resource without namespace should be rejected")
	}
	if err := validateResourceInput(&ResourceInput{Kind: "Host", Name: "api", Address: "10.0.0.1"}); err == nil {
		t.Fatal("host resource without an authoritative host relation should be rejected")
	}
	if err := validateComponentInput(&ComponentInput{Category: "database", Type: "MySQL", Name: "orders", Port: 3306}); err == nil {
		t.Fatal("component without address should be rejected")
	}
}

func TestValidateAssetInputsAcceptCompleteRecords(t *testing.T) {
	if err := validateDomainInput(&DomainInput{Domain: "api.example.com", Protocol: "https", Port: 443, Path: "/"}); err != nil {
		t.Fatalf("valid domain rejected: %v", err)
	}
	if err := validateResourceInput(&ResourceInput{Kind: "Deployment", Name: "api", ClusterID: 2, Namespace: "prod"}); err != nil {
		t.Fatalf("valid kubernetes resource rejected: %v", err)
	}
	if err := validateComponentInput(&ComponentInput{Category: "database", Type: "MySQL", Name: "orders", Address: "mysql.internal", Port: 3306}); err != nil {
		t.Fatalf("valid component rejected: %v", err)
	}
}

func TestValidateDomainInputNormalizesServerOwnedFields(t *testing.T) {
	in := &DomainInput{Domain: " API.Example.COM. ", Protocol: " HTTPS ", Port: 443, Path: " /health ", CertificateID: 8}
	if err := validateDomainInput(in); err != nil {
		t.Fatalf("valid domain rejected: %v", err)
	}
	if in.Domain != "api.example.com" || in.Protocol != "https" || in.Path != "/health" || in.CertificateID != 8 {
		t.Fatalf("domain was not normalized: %#v", in)
	}

	in.Protocol = "http"
	if err := validateDomainInput(in); err != nil {
		t.Fatalf("valid HTTP domain rejected: %v", err)
	}
	if in.CertificateID != 0 {
		t.Fatal("non-HTTPS domain retained a certificate relation")
	}
}

func TestValidateDomainInputRejectsInvalidDomain(t *testing.T) {
	if err := validateDomainInput(&DomainInput{Domain: "https://api.example.com", Protocol: "https", Port: 443, Path: "/"}); err == nil {
		t.Fatal("domain containing a scheme should be rejected")
	}
}

func TestValidateDomainInputAcceptsInternalHostnameAndIPAddress(t *testing.T) {
	for _, domain := range []string{"order-api", "order-api.service.local", "10.122.24.32"} {
		in := &DomainInput{Domain: domain, Protocol: "http", Port: 8080, Path: "/health"}
		if err := validateDomainInput(in); err != nil {
			t.Fatalf("valid internal host %q rejected: %v", domain, err)
		}
	}
}

func TestValidateDomainInputRejectsIPv6UntilURLFormattingSupportsIt(t *testing.T) {
	in := &DomainInput{Domain: "2001:db8::1", Protocol: "http", Port: 8080, Path: "/health"}
	if err := validateDomainInput(in); err == nil {
		t.Fatal("IPv6 domain should be rejected until bracketed endpoint formatting is supported")
	}
}

func TestValidateDependencyRejectsApplicationSelfReference(t *testing.T) {
	in := &DependencyInput{SourceApplicationID: 7, TargetApplicationID: 7, Protocol: "HTTP", Endpoint: "http://self"}
	if err := validateDependency(in); err == nil {
		t.Fatal("application self dependency should be rejected")
	}
}

func TestValidateDependencyClearsStaleTargetNameForManagedTarget(t *testing.T) {
	in := &DependencyInput{SourceApplicationID: 7, TargetApplicationID: 8, TargetName: "stale", Protocol: "HTTP", Endpoint: "http://target"}
	if err := validateDependency(in); err != nil {
		t.Fatalf("valid dependency rejected: %v", err)
	}
	if in.TargetName != "" {
		t.Fatalf("managed target retained stale target name: %q", in.TargetName)
	}
}

func TestValidateAssetInputsRejectOversizedFields(t *testing.T) {
	long := make([]byte, 181)
	for i := range long {
		long[i] = 'a'
	}
	if err := validateResourceInput(&ResourceInput{Kind: "Other", Name: string(long), Address: "service.internal"}); err == nil {
		t.Fatal("oversized resource name should be rejected")
	}
	if err := validateComponentInput(&ComponentInput{Category: "database", Type: "MySQL", Name: string(long), Address: "mysql.internal", Port: 3306}); err == nil {
		t.Fatal("oversized component name should be rejected")
	}
}

func TestDNSProviderFromNameservers(t *testing.T) {
	tests := []struct {
		nameservers []string
		want        string
	}{
		{[]string{"ns1.alidns.com", "ns2.alidns.com"}, "阿里云 DNS"},
		{[]string{"ada.ns.cloudflare.com"}, "Cloudflare"},
		{[]string{"ns-100.awsdns-12.com"}, "AWS Route 53"},
		{[]string{"ns1.internal.example"}, "ns1.internal.example"},
	}
	for _, test := range tests {
		if got := dnsProviderFromNameservers(test.nameservers); got != test.want {
			t.Fatalf("provider mismatch: got %q, want %q", got, test.want)
		}
	}
}

func TestAggregateHealthUsesWorstObservedState(t *testing.T) {
	status, _ := aggregateHealth([]string{"healthy", "warning", "unknown"})
	if status != "warning" {
		t.Fatalf("expected warning, got %s", status)
	}
	status, _ = aggregateHealth([]string{"healthy", "unhealthy"})
	if status != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", status)
	}
	status, _ = aggregateHealth(nil)
	if status != "unknown" {
		t.Fatalf("expected unknown for empty assets, got %s", status)
	}
}

func TestLifecycleFollowsEnvironment(t *testing.T) {
	for _, kind := range []string{"production", "staging", "test", "development"} {
		if lifecycleFromEnvironment(kind) != kind {
			t.Fatalf("environment kind %s was not preserved", kind)
		}
	}
}
