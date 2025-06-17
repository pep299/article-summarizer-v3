# Project Context

## Project Overview
- Go-based RSS article summarizer system (Reddit, Hatena, Lobsters support)
- Integrates with GCS, Gemini API, and Slack
- Implements feed-specific strategy pattern for extensibility

## Technical Architecture
- Standard Go directory structure (`internal/`, `cmd/`)
- Shared mocks in `internal/mocks/` following Go conventions
- Feed-specific processors using strategy pattern
- 1:1 test-to-implementation correspondence

## Development Practices
- Follow Go best practices and idioms
- Use context for external API calls, not for in-memory operations
- Implement comprehensive error handling and logging
- Maintain clean separation between transport, service, and repository layers

## Communication Preferences
- For "〜って何？" questions: Provide detailed explanations with background context in Japanese
- For technical discussions: Use specific implementation details rather than summaries
- For diff reviews: Explain actual changes and reasoning, not just what was modified
- For refactoring: Always explain the "why" behind architectural decisions

## Current Focus Areas
- Feed processing optimization and error handling
- Test coverage and mock implementation consistency
- GCS repository pattern and key generation strategies
- Context usage patterns in Go applications