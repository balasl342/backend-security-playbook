package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validCustomer() Customer {
	return Customer{
		Name:       "Ada Lovelace",
		Email:      "ada@example.com",
		Phone:      "555-0100",
		SSN:        "123-45-6789",
		CreditCard: "4111111111111111",
		Address:    "1 Analytical Engine Way",
	}
}

func TestCustomer_Validate_ValidPasses(t *testing.T) {
	assert.NoError(t, validCustomer().Validate())
}

func TestCustomer_Validate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Customer)
		wantMsg string
	}{
		{"missing name", func(c *Customer) { c.Name = "" }, "name"},
		{"missing email", func(c *Customer) { c.Email = "" }, "email"},
		{"missing ssn", func(c *Customer) { c.SSN = "" }, "ssn"},
		{"missing credit card", func(c *Customer) { c.CreditCard = "" }, "credit_card"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validCustomer()
			tt.mutate(&c)

			err := c.Validate()
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidCustomer))
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestCustomer_Validate_InvalidEmail(t *testing.T) {
	c := validCustomer()
	c.Email = "not-an-email"

	err := c.Validate()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCustomer))
}

func TestCustomer_Validate_MultipleMissingFieldsListed(t *testing.T) {
	c := Customer{}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "ssn")
	assert.Contains(t, err.Error(), "credit_card")
}
