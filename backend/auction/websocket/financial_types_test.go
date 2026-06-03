package websocket

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestWebSocketFinancialFieldsUseDecimal(t *testing.T) {
	decimalType := reflect.TypeOf(decimal.Decimal{})

	for _, tc := range []struct {
		model any
		field string
	}{
		{BidPlacedData{}, "Amount"},
		{BidPlacedData{}, "CurrentPrice"},
		{RankItem{}, "Amount"},
		{OvertakenData{}, "NewPrice"},
		{AuctionEndedData{}, "FinalPrice"},
		{SyncResponseData{}, "CurrentPrice"},
		{SkyLampActivatedData{}, "InitialBidAmount"},
		{SkyLampActivatedData{}, "MaxPriceLimit"},
		{SkyLampAutoBidData{}, "Amount"},
		{SkyLampAutoBidData{}, "RemainingBudget"},
	} {
		field, ok := reflect.TypeOf(tc.model).FieldByName(tc.field)
		if !ok {
			t.Fatalf("missing field %s", tc.field)
		}
		if field.Type != decimalType {
			t.Fatalf("%T.%s should use decimal.Decimal, got %s", tc.model, tc.field, field.Type)
		}
	}
}
