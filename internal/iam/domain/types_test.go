package domain

import "testing"

func TestResourceMatchHonorsSegmentBoundary(t *testing.T) {
	if !ResourceMatch("organizations/acme", "organizations/acme/products/p1") {
		t.Fatal("organization binding should match descendant")
	}
	if ResourceMatch("organizations/acme", "organizations/acme-other/products/p1") {
		t.Fatal("organization binding must not match a prefix collision")
	}
	if ResourceMatch("organizations/acme/products/p1", "organizations/acme/products/p2") {
		t.Fatal("resource binding must not match a sibling")
	}
}

func TestParseResource(t *testing.T) {
	for _, name := range []string{
		"organizations/acme",
		"organizations/acme/products/8db144b3-a149-4e67-83eb-38d5c7bc1ef5",
		"organizations/acme/serviceAccounts/8db144b3-a149-4e67-83eb-38d5c7bc1ef5",
	} {
		if _, err := ParseResource(name); err != nil {
			t.Fatalf("ParseResource(%q): %v", name, err)
		}
	}
	for _, name := range []string{
		"organizations/Acme",
		"organizations/acme/unknown/id",
		"organizations/acme/products",
		"organizations/acme/products/id/children/id",
	} {
		if _, err := ParseResource(name); err == nil {
			t.Fatalf("ParseResource(%q) unexpectedly succeeded", name)
		}
	}
}

func TestResourceBuilders(t *testing.T) {
	organization, err := OrganizationResource("acme")
	if err != nil || organization.Name != "organizations/acme" {
		t.Fatalf("unexpected organization resource: %+v, %v", organization, err)
	}
	account, err := OrganizationChildResource("acme", ResourceServiceAccounts, "account-1")
	if err != nil || account.Name != "organizations/acme/serviceAccounts/account-1" {
		t.Fatalf("unexpected child resource: %+v, %v", account, err)
	}
	if _, err := OrganizationResource("invalid slug"); err == nil {
		t.Fatal("expected invalid organization slug to fail")
	}
}

func TestPrincipalKeysSeparateIssuerAndType(t *testing.T) {
	user := Principal{Type: PrincipalUser, Issuer: "issuer-a", Subject: "same"}
	otherIssuer := Principal{Type: PrincipalUser, Issuer: "issuer-b", Subject: "same"}
	serviceAccount := Principal{Type: PrincipalServiceAccount, Issuer: "issuer-a", Subject: "same"}
	if user.Key() == otherIssuer.Key() || user.Key() == serviceAccount.Key() {
		t.Fatal("principal keys must separate issuer and principal type")
	}
}
