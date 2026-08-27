package contracts_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestContractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		contract contracts.BaseContract
		cap      lifecycle.CapabilityKind
	}{
		{"RepositoryManagementContract", contracts.DefaultRepositoryContractMetadata(), lifecycle.CapabilityRepository},
		{"FileContract", contracts.DefaultFileContractMetadata(), lifecycle.CapabilityRepository},
		{"PackageContract", contracts.DefaultPackageContractMetadata(), lifecycle.CapabilityRepository},
		{"SymbolContract", contracts.DefaultSymbolContractMetadata(), lifecycle.CapabilitySymbol},
		{"GraphContract", contracts.DefaultGraphContractMetadata(), lifecycle.CapabilityGraph},
		{"SearchContract", contracts.DefaultSearchContractMetadata(), lifecycle.CapabilitySearch},
		{"IntelligenceContract", contracts.DefaultIntelligenceContractMetadata(), lifecycle.CapabilityIntelligence},
		{"AnalysisContract", contracts.DefaultAnalysisContractMetadata(), lifecycle.CapabilityIntelligence},
		{"NavigationContract", contracts.DefaultNavigationContractMetadata(), lifecycle.CapabilityIntelligence},
		{"ReasoningContract", contracts.DefaultReasoningContractMetadata(), lifecycle.CapabilityIntelligence},
		{"EventContract", contracts.DefaultEventContractMetadata(), lifecycle.CapabilityIntelligence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.contract.Name() != tt.name {
				t.Errorf("expected contract name %q, got %q", tt.name, tt.contract.Name())
			}
			if tt.contract.Capability() != tt.cap {
				t.Errorf("expected capability %q, got %q", tt.cap, tt.contract.Capability())
			}
			if tt.contract.Lifecycle() != lifecycle.StateSupported {
				t.Errorf("expected StateSupported, got %v", tt.contract.Lifecycle())
			}
			if err := tt.contract.Validate(); err != nil {
				t.Errorf("expected valid contract, got error: %v", err)
			}
		})
	}
}

func TestPaginationOptions(t *testing.T) {
	opts := contracts.PaginationOptions{Offset: -5, Limit: 0}
	norm := opts.Normalize(50, 100)
	if norm.Offset != 0 {
		t.Errorf("expected offset 0, got %d", norm.Offset)
	}
	if norm.Limit != 50 {
		t.Errorf("expected limit 50, got %d", norm.Limit)
	}

	opts.Limit = 200
	norm = opts.Normalize(50, 100)
	if norm.Limit != 100 {
		t.Errorf("expected capped limit 100, got %d", norm.Limit)
	}
}

func TestValidateContract(t *testing.T) {
	valid := contracts.DefaultRepositoryContractMetadata()
	if err := contracts.ValidateContract(valid); err != nil {
		t.Errorf("expected valid contract, got error: %v", err)
	}

	invalid := contracts.NewBaseContract("", "", version.SemVer{}, lifecycle.StateSupported, "")
	if err := contracts.ValidateContract(invalid); err == nil {
		t.Errorf("expected error for invalid contract")
	}
}
