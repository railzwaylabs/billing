package domain

import "context"

type Currency struct {
	Code, Name, Symbol string
	MinorUnit          int
	Active             bool
}
type MeasurementUnit struct {
	Code, Name, Symbol, Category, Description string
	Active                                    bool
}

type ReferenceRepository interface {
	ListCurrencies(context.Context) ([]Currency, error)
	ListMeasurementUnits(context.Context) ([]MeasurementUnit, error)
}
