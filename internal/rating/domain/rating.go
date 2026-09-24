package domain

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	catalogue "github.com/railzwaylabs/billing/internal/catalogue/domain"
	meterdomain "github.com/railzwaylabs/billing/internal/meter/domain"
	shareddomain "github.com/railzwaylabs/billing/internal/shared/domain"
	usage "github.com/railzwaylabs/billing/internal/usage/domain"
)

type TierBreakdown struct {
	StartQuantity shareddomain.Quantity
	Quantity      shareddomain.Quantity
	UnitAmount    shareddomain.Money
	Amount        shareddomain.Money
}

type Result struct {
	OrganizationID     uuid.UUID
	CustomerID         uuid.UUID
	SubscriptionID     uuid.UUID
	SubscriptionItemID uuid.UUID
	ProductID          uuid.UUID
	PriceID            uuid.UUID
	PriceChargeID      uuid.UUID
	MeterID            uuid.UUID
	PeriodStart        time.Time
	PeriodEnd          time.Time
	UsageQuantity      shareddomain.Quantity
	PricingUnit        shareddomain.Quantity
	Unit               string
	Amount             shareddomain.Money
	Tiers              []TierBreakdown
}

func Aggregate(aggregation meterdomain.Aggregation, events []usage.Event) (shareddomain.Quantity, error) {
	switch aggregation {
	case meterdomain.AggregationCount:
		return shareddomain.WholeQuantity(int64(len(events)))
	case meterdomain.AggregationSum:
		total := shareddomain.Quantity{}
		for _, event := range events {
			var err error
			total, err = total.Add(event.Value)
			if err != nil {
				return shareddomain.Quantity{}, err
			}
		}
		return total, nil
	default:
		return shareddomain.Quantity{}, fmt.Errorf("unsupported aggregation %q", aggregation)
	}
}

// Calculate applies graduated tiers. Each tier only prices usage within its own range.
func Calculate(quantity shareddomain.Quantity, currency string, charge catalogue.PriceCharge) (shareddomain.Money, []TierBreakdown, error) {
	if err := quantity.Validate(); err != nil {
		return shareddomain.Money{}, nil, err
	}

	if len(charge.Tiers) == 0 || charge.UnitQuantity.Micros <= 0 {
		return shareddomain.Money{}, nil, fmt.Errorf("price has no valid tiers")
	}

	tiers := append([]catalogue.ChargeTier(nil), charge.Tiers...)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].StartQuantity.Micros < tiers[j].StartQuantity.Micros
	})

	if tiers[0].StartQuantity.Micros != 0 {
		return shareddomain.Money{}, nil, fmt.Errorf("first tier must start at zero")
	}

	total, err := shareddomain.NewMoney(currency, 0)
	if err != nil {
		return shareddomain.Money{}, nil, err
	}

	breakdown := make([]TierBreakdown, 0, len(tiers))
	for i, tier := range tiers {
		start := tier.StartQuantity.Micros
		if quantity.Micros <= start {
			break
		}

		end := quantity.Micros
		if i+1 < len(tiers) && tiers[i+1].StartQuantity.Micros < end {
			end = tiers[i+1].StartQuantity.Micros
		}

		band := shareddomain.Quantity{Micros: end - start}
		amount, err := shareddomain.Price(band, charge.UnitQuantity, tier.UnitAmount)
		if err != nil {
			return shareddomain.Money{}, nil, err
		}

		total, err = total.Add(amount)
		if err != nil {
			return shareddomain.Money{}, nil, err
		}

		breakdown = append(breakdown, TierBreakdown{
			StartQuantity: tier.StartQuantity,
			Quantity:      band,
			UnitAmount:    tier.UnitAmount,
			Amount:        amount,
		})
	}
	return total, breakdown, nil
}
