package domain

import (
	"fmt"
	"math/big"
	"regexp"
)

const (
	QuantityScale int64 = 1_000_000
	MoneyScale    int64 = 1_000_000_000
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Quantity stores usage in millionths of a unit.
type Quantity struct {
	Micros int64
}

func NewQuantity(micros int64) (Quantity, error) {
	if micros < 0 {
		return Quantity{}, fmt.Errorf("quantity must not be negative")
	}
	return Quantity{Micros: micros}, nil
}

func WholeQuantity(value int64) (Quantity, error) {
	if value < 0 || value > (int64(^uint64(0)>>1)/QuantityScale) {
		return Quantity{}, fmt.Errorf("quantity is out of range")
	}
	return Quantity{Micros: value * QuantityScale}, nil
}

func (q Quantity) Validate() error {
	if q.Micros < 0 {
		return fmt.Errorf("quantity must not be negative")
	}
	return nil
}

func (q Quantity) Add(other Quantity) (Quantity, error) {
	result := new(big.Int).Add(big.NewInt(q.Micros), big.NewInt(other.Micros))
	if !result.IsInt64() {
		return Quantity{}, fmt.Errorf("quantity overflow")
	}
	return NewQuantity(result.Int64())
}

type Money struct {
	Currency string
	Nanos    int64
}

func NewMoney(currency string, nanos int64) (Money, error) {
	if !currencyPattern.MatchString(currency) {
		return Money{}, fmt.Errorf("currency must be a three-letter uppercase code")
	}
	if nanos < 0 {
		return Money{}, fmt.Errorf("money must not be negative")
	}
	return Money{Currency: currency, Nanos: nanos}, nil
}

func (m Money) Validate() error {
	_, err := NewMoney(m.Currency, m.Nanos)
	return err
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s and %s", m.Currency, other.Currency)
	}
	result := new(big.Int).Add(big.NewInt(m.Nanos), big.NewInt(other.Nanos))
	if !result.IsInt64() {
		return Money{}, fmt.Errorf("money overflow")
	}
	return NewMoney(m.Currency, result.Int64())
}

// Price computes quantity/unitQuantity*unitPrice using half-up rounding to one nano.
func Price(quantity, unitQuantity Quantity, unitPrice Money) (Money, error) {
	if err := quantity.Validate(); err != nil {
		return Money{}, err
	}
	if err := unitQuantity.Validate(); err != nil || unitQuantity.Micros == 0 {
		return Money{}, fmt.Errorf("pricing unit quantity must be positive")
	}
	if err := unitPrice.Validate(); err != nil {
		return Money{}, err
	}

	numerator := new(big.Int).Mul(big.NewInt(quantity.Micros), big.NewInt(unitPrice.Nanos))
	denominator := big.NewInt(unitQuantity.Micros)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return Money{}, fmt.Errorf("priced amount overflow")
	}
	return NewMoney(unitPrice.Currency, quotient.Int64())
}
