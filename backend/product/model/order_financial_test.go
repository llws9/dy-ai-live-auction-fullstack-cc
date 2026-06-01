package model

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOrderFinalPriceUsesDecimal(t *testing.T) {
	field, ok := reflect.TypeOf(Order{}).FieldByName("FinalPrice")

	assert.True(t, ok)
	assert.Equal(t, reflect.TypeOf(decimal.Decimal{}), field.Type)
}
