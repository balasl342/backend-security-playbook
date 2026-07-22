package domain

import "errors"

// Sentinel errors returned by repositories and services. Callers use
// errors.Is to branch on these regardless of which layer produced them.
var (
	// ErrCustomerNotFound is returned when a customer id has no matching row.
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrInvalidCustomer is returned when a Customer fails validation before
	// being persisted.
	ErrInvalidCustomer = errors.New("invalid customer")
)
