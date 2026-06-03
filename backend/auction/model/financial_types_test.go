package model

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAuctionFinancialFieldsUseDecimal(t *testing.T) {
	decimalType := reflect.TypeOf(decimal.Decimal{})

	for _, tc := range []struct {
		model any
		field string
	}{
		{Auction{}, "CurrentPrice"},
		{AuctionRule{}, "StartPrice"},
		{AuctionRule{}, "Increment"},
		{Bid{}, "Amount"},
	} {
		field, ok := reflect.TypeOf(tc.model).FieldByName(tc.field)
		assert.True(t, ok, "missing field %s", tc.field)
		assert.Equal(t, decimalType, field.Type, "%s should use decimal.Decimal", tc.field)
	}
}
