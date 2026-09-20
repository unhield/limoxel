package security

import "testing"

func TestTrustState(t *testing.T) {
	if !TrustTrusted.IsAcceptable() {
		t.Fatal("expected TrustTrusted to be acceptable")
	}
	if !TrustVerified.IsAcceptable() {
		t.Fatal("expected TrustVerified to be acceptable")
	}
	if TrustUnverified.IsAcceptable() {
		t.Fatal("expected TrustUnverified not to be acceptable")
	}
	if TrustRejected.IsAcceptable() {
		t.Fatal("expected TrustRejected not to be acceptable")
	}
	if TrustTrusted.String() != "TRUSTED" {
		t.Fatalf("expected string TRUSTED, got %s", TrustTrusted.String())
	}
}

func TestDomainAndPermissions(t *testing.T) {
	if DomainRepository != "repository" {
		t.Fatalf("expected repository, got %s", DomainRepository)
	}
	if PermRepoRead != "repository.read" {
		t.Fatalf("expected repository.read, got %s", PermRepoRead)
	}
	if PermFileWrite != "file.write" {
		t.Fatalf("expected file.write, got %s", PermFileWrite)
	}
	if PermNetworkOutbound != "network.outbound" {
		t.Fatalf("expected network.outbound, got %s", PermNetworkOutbound)
	}
}
