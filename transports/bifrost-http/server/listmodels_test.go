package server

import (
	"context"
	"testing"
	"time"

	bifrost "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/grant"
	"github.com/maximhq/bifrost/transports/bifrost-http/lib"
)

func TestListModelsProvidersAllowedByIncludesConfiguredProvidersForAllowAllPermit(t *testing.T) {
	permit := grant.NewPermit(grant.PermitVirtualKey, "vk-test", "Test VK", true, false, nil, nil,
		grant.WithAllowAllProviders(true))
	access := grant.NewAccess([]schemas.Permit{permit}, nil, "", nil)
	configured := []schemas.ModelProvider{schemas.OpenAI, schemas.Anthropic}

	got := listModelsProvidersAllowedBy(access, configured)
	if len(got) != len(configured) {
		t.Fatalf("providers = %#v, want %#v", got, configured)
	}
	for i := range configured {
		if got[i] != configured[i] {
			t.Fatalf("providers = %#v, want %#v", got, configured)
		}
	}
}

func TestListModelsProvidersAllowedByKeepsExplicitProviderRestriction(t *testing.T) {
	permit := grant.NewPermit(grant.PermitVirtualKey, "vk-test", "Test VK", true, false,
		[]schemas.ProviderPermit{{Provider: string(schemas.OpenAI), AllowedModels: []string{"*"}}}, nil)
	access := grant.NewAccess([]schemas.Permit{permit}, nil, "", nil)

	got := listModelsProvidersAllowedBy(access, []schemas.ModelProvider{schemas.OpenAI, schemas.Anthropic})
	if len(got) != 1 || got[0] != schemas.OpenAI {
		t.Fatalf("providers = %#v, want %#v", got, []schemas.ModelProvider{schemas.OpenAI})
	}
}

// A deployment with no governance resolves no access, and the listing must keep the fan-out it
// already had. Publishing an empty list here would mean "no provider may serve this", turning an
// unrestricted listing into one that lists nothing, so this pins the guard rather than the rule.
func TestNarrowListModelsProvidersLeavesFanOutAloneWithoutGovernance(t *testing.T) {
	// ResolveAccess warns through the package logger when no governance plugin is registered,
	// which is nil until a server sets it.
	SetLogger(bifrost.NewNoOpLogger())

	// Ctx is what the governance plugin is looked up through, and a real server always has one.
	s := &BifrostHTTPServer{
		Config: &lib.Config{},
		Ctx:    schemas.NewBifrostContext(context.Background(), time.Time{}),
	}
	bifrostCtx := schemas.NewBifrostContext(context.Background(), time.Time{})

	s.NarrowListModelsProviders(bifrostCtx)

	if got := bifrostCtx.Value(schemas.BifrostContextKeyAvailableProviders); got != nil {
		t.Fatalf("expected nothing to be published, got %#v", got)
	}
}

// Narrowing runs on the request path, so a context it was never given must not take the listing
// down with it.
func TestNarrowListModelsProvidersWithoutContext(t *testing.T) {
	SetLogger(bifrost.NewNoOpLogger())

	s := &BifrostHTTPServer{
		Config: &lib.Config{},
		Ctx:    schemas.NewBifrostContext(context.Background(), time.Time{}),
	}

	s.NarrowListModelsProviders(nil)
}

func TestFilterListModelsDelegatesOnlyWhenConfigured(t *testing.T) {
	response := &schemas.BifrostListModelsResponse{Data: []schemas.Model{{ID: "openai/model-a"}}}
	server := &BifrostHTTPServer{}
	if err := server.FilterListModels(nil, response); err != nil {
		t.Fatalf("unconfigured filter returned error: %v", err)
	}

	calls := 0
	server.ListModelsFilter = func(_ *schemas.BifrostContext, got *schemas.BifrostListModelsResponse) error {
		calls++
		got.Data = nil
		return nil
	}
	if err := server.FilterListModels(nil, response); err != nil {
		t.Fatalf("configured filter returned error: %v", err)
	}
	if calls != 1 || len(response.Data) != 0 {
		t.Fatalf("filter calls = %d, response = %#v", calls, response.Data)
	}
}
