package imagegen

import (
	"log/slog"
)

// ManagerOption configures the Manager.
type ManagerOption func(*Manager)

// WithLogger sets a structured logger for the manager.
func WithLogger(logger *slog.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// WithStorage sets a storage backend for persisting generated images.
func WithStorage(storage Storage) ManagerOption {
	return func(m *Manager) {
		m.storage = storage
	}
}

// WithDefaultModel sets the default model used when config.Model is empty.
func WithDefaultModel(model Model) ManagerOption {
	return func(m *Manager) {
		m.defaultModel = model
	}
}

// NewManager creates a Manager with the given providers and options.
//
// Example:
//
//	gen, err := gemini.NewWithAPIKey(ctx, apiKey)
//	if err != nil {
//	    return err
//	}
//	manager := imagegen.NewManager(gen)
//
// With options:
//
//	manager := imagegen.NewManager(gen,
//	    imagegen.WithLogger(slog.Default()),
//	    imagegen.WithDefaultModel(imagegen.ModelNanoBanana1),
//	)
func NewManager(defaultProvider ImageGenerator, opts ...ManagerOption) *Manager {
	m := New()

	models := defaultProvider.Models()
	for i := range models {
		info := &models[i]

		m.providers[info.Provider] = defaultProvider

		m.RegisterModel(Model(info.Name),
			ModelMapping{
				Provider:        info.Provider,
				ActualModelName: info.APIModelName,
			},
			info)
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
