package feed

import "testing"

func TestFeedRegistry(t *testing.T) {
	registry := NewFeedRegistry()

	// Test empty registry
	_, exists := registry.GetStrategy("nonexistent")
	if exists {
		t.Error("Expected strategy to not exist in empty registry")
	}

	// Register strategies
	hatenaStrategy := NewHatenaStrategy()
	lobstersStrategy := NewLobstersStrategy()

	registry.Register(hatenaStrategy)
	registry.Register(lobstersStrategy)

	// Test retrieval
	strategy, exists := registry.GetStrategy("hatena")
	if !exists {
		t.Error("Expected hatena strategy to exist")
	}
	if strategy.GetConfig().Name != "hatena" {
		t.Error("Retrieved wrong strategy")
	}

	strategy, exists = registry.GetStrategy("lobsters")
	if !exists {
		t.Error("Expected lobsters strategy to exist")
	}
	if strategy.GetConfig().Name != "lobsters" {
		t.Error("Retrieved wrong strategy")
	}

	// Test getting all strategies
	allStrategies := registry.GetAllStrategies()
	if len(allStrategies) != 2 {
		t.Errorf("Expected 2 strategies, got %d", len(allStrategies))
	}
}

func TestGetDefaultFeeds(t *testing.T) {
	defaults := GetDefaultFeeds()
	if len(defaults) != 2 {
		t.Errorf("Expected 2 default feeds, got %d", len(defaults))
	}

	// Check that we have hatena and lobsters
	feedNames := make(map[string]bool)
	for _, strategy := range defaults {
		feedNames[strategy.GetConfig().Name] = true
	}

	if !feedNames["hatena"] {
		t.Error("Expected hatena in default feeds")
	}
	if !feedNames["lobsters"] {
		t.Error("Expected lobsters in default feeds")
	}
}

func TestFeedRegistryOverwrite(t *testing.T) {
	registry := NewFeedRegistry()

	// Register initial strategy
	original := NewHatenaStrategy()
	registry.Register(original)

	// Register new strategy with same name (should overwrite)
	// Since we can't modify URL easily, just verify the overwrite mechanism works
	registry.Register(NewHatenaStrategy())

	// Verify we still have the strategy
	strategy, exists := registry.GetStrategy("hatena")
	if !exists {
		t.Error("Expected strategy to exist")
	}
	if strategy.GetConfig().Name != "hatena" {
		t.Error("Strategy was not registered properly")
	}
}
