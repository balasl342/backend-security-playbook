// Package repository defines persistence interfaces for domain entities.
// Concrete implementations (plaintext, envelope-encrypted, in-memory for
// tests) live alongside this file and are selected at wiring time based on
// configuration.
package repository

import (
	"context"

	"github.com/balac/backend-security-playground/internal/domain"
)

// CustomerRepository persists and retrieves Customer records. Implementations
// are responsible for any at-rest encryption of sensitive fields; callers
// always deal in plaintext domain.Customer values.
type CustomerRepository interface {
	// Create inserts a new customer and returns the stored record, including
	// any server-generated fields (ID, timestamps).
	Create(ctx context.Context, customer domain.Customer) (domain.Customer, error)

	// GetByID fetches a customer by id. Returns domain.ErrCustomerNotFound if
	// no matching row exists.
	GetByID(ctx context.Context, id string) (domain.Customer, error)
}
