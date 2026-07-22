// Package domain contains core business entities, free of any persistence,
// transport, or encryption concerns. A Customer here always holds plaintext
// values in memory - encryption is applied at the repository boundary when
// writing to storage, and reversed when reading, so the rest of the
// application never has to reason about ciphertext.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// Customer is a single customer record.
type Customer struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	SSN        string    `json:"ssn"`
	CreditCard string    `json:"credit_card"`
	Address    string    `json:"address"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Validate checks that required fields are present and well-formed. It
// wraps ErrInvalidCustomer so callers can distinguish validation failures
// from other errors with errors.Is.
func (c Customer) Validate() error {
	var missing []string

	if strings.TrimSpace(c.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(c.Email) == "" {
		missing = append(missing, "email")
	} else if !strings.Contains(c.Email, "@") {
		return fmt.Errorf("%w: email is not a valid address", ErrInvalidCustomer)
	}
	if strings.TrimSpace(c.SSN) == "" {
		missing = append(missing, "ssn")
	}
	if strings.TrimSpace(c.CreditCard) == "" {
		missing = append(missing, "credit_card")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: missing required field(s): %s", ErrInvalidCustomer, strings.Join(missing, ", "))
	}

	return nil
}

// SensitiveFields are the customer fields that Mode B encrypts at rest.
// Name, Email, and Phone are stored in the clear even in Mode B - they are
// used for lookups/search and are not classified as highly sensitive in
// this playground's threat model.
const (
	FieldSSN        = "ssn"
	FieldCreditCard = "credit_card"
	FieldAddress    = "address"
)
