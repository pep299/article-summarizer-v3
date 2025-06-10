package feed

// FeedRegistry manages available feed strategies
type FeedRegistry struct {
	strategies map[string]FeedStrategy
}

func NewFeedRegistry() *FeedRegistry {
	return &FeedRegistry{
		strategies: make(map[string]FeedStrategy),
	}
}

func (r *FeedRegistry) Register(strategy FeedStrategy) {
	config := strategy.GetConfig()
	r.strategies[config.Name] = strategy
}

func (r *FeedRegistry) GetStrategy(feedName string) (FeedStrategy, bool) {
	strategy, exists := r.strategies[feedName]
	return strategy, exists
}

func (r *FeedRegistry) GetAllStrategies() map[string]FeedStrategy {
	return r.strategies
}

// GetDefaultFeeds returns all default feed configurations
func GetDefaultFeeds() []FeedStrategy {
	return []FeedStrategy{
		NewHatenaStrategy(),
		NewLobstersStrategy(),
	}
}
