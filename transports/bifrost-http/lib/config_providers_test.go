package lib

import (
	"context"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/configstore"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type providerLoadStore struct {
	configstore.ConfigStore
	providers   map[schemas.ModelProvider]configstore.ProviderConfig
	updateCalls int
}

func (s *providerLoadStore) GetProvidersConfig(context.Context) (map[schemas.ModelProvider]configstore.ProviderConfig, error) {
	return s.providers, nil
}

func (s *providerLoadStore) UpdateProvidersConfig(context.Context, map[schemas.ModelProvider]configstore.ProviderConfig, ...*gorm.DB) error {
	s.updateCalls++
	return nil
}

func TestLoadProvidersDoesNotRewriteDatabaseOnlyConfiguration(t *testing.T) {
	initTestLogger()
	store := &providerLoadStore{providers: map[schemas.ModelProvider]configstore.ProviderConfig{
		schemas.OpenAI: {
			Keys: []schemas.Key{{ID: "key-1", Value: *schemas.NewSecretVar("sk-test")}},
		},
	}}
	config := &Config{ConfigStore: store}

	require.NoError(t, loadProviders(context.Background(), config, &ConfigData{SourceOfTruth: SourceOfTruthSplit}))
	require.Zero(t, store.updateCalls)
	require.Equal(t, store.providers, config.Providers)
}
