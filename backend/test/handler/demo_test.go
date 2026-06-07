package handler

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shopspring/decimal"
)

func TestComputeFollowBidAmount(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		incr     string
		override string
		want     string
	}{
		{"override wins", "100", "10", "500", "500"},
		{"current plus increment", "100", "10", "", "110"},
		{"zero current uses increment", "0", "5", "", "5"},
		{"empty increment defaults to 1", "100", "", "", "101"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur := mustFollowBidAmount(t, c.current)
			incr := zeroFollowBidAmount()
			if c.incr != "" {
				incr = mustFollowBidAmount(t, c.incr)
			}
			var override *decimal.Decimal
			if c.override != "" {
				v := mustFollowBidAmount(t, c.override)
				override = &v
			}
			got := computeFollowBidAmount(cur, incr, override)
			want := mustFollowBidAmount(t, c.want)
			if !got.Equal(want) {
				t.Fatalf("computeFollowBidAmount(%s,%s,%v)=%s want %s", c.current, c.incr, override, got, want)
			}
		})
	}
}

func TestValidateRechargeRequest(t *testing.T) {
	cases := []struct {
		name    string
		userID  int64
		amount  string
		wantErr bool
	}{
		{"valid buyer B", buyerBUserID, "100.00", false},
		{"reject other demo user", buyerAUserID, "100.00", true},
		{"zero user", 0, "100.00", true},
		{"empty amount", buyerBUserID, "", true},
		{"non-positive amount", buyerBUserID, "0", true},
		{"negative amount", buyerBUserID, "-5", true},
		{"bad amount", buyerBUserID, "abc", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRechargeRequest(c.userID, c.amount)
			if (err != nil) != c.wantErr {
				t.Fatalf("validateRechargeRequest(%d,%q) err=%v wantErr=%v", c.userID, c.amount, err, c.wantErr)
			}
		})
	}
}

func TestDemoUserIDFromAuthorization(t *testing.T) {
	const secret = "demo-secret"
	cases := []struct {
		name    string
		header  string
		secret  string
		want    int64
		wantErr bool
	}{
		{"valid demo buyer", "Bearer " + signDemoToken(t, secret, buyerAUserID), secret, buyerAUserID, false},
		{"valid demo admin", "Bearer " + signDemoToken(t, secret, adminUserID), secret, adminUserID, false},
		{"missing bearer", signDemoToken(t, secret, buyerAUserID), secret, 0, true},
		{"bad secret", "Bearer " + signDemoToken(t, secret, buyerAUserID), "other-secret", 0, true},
		{"non demo user", "Bearer " + signDemoToken(t, secret, 42), secret, 0, true},
		{"empty configured secret", "Bearer " + signDemoToken(t, secret, buyerAUserID), "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := demoUserIDFromAuthorization(c.header, c.secret)
			if (err != nil) != c.wantErr {
				t.Fatalf("demoUserIDFromAuthorization() err=%v wantErr=%v", err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("demoUserIDFromAuthorization()=%d want %d", got, c.want)
			}
		})
	}
}

func TestDecimalToBidAmountRejectsUnsupportedRange(t *testing.T) {
	_, err := decimalToBidAmount(decimal.New(1, 400))
	if err == nil {
		t.Fatalf("decimalToBidAmount() expected range error")
	}
}

func mustFollowBidAmount(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	amount, err := parseFollowBidAmount(raw)
	if err != nil {
		t.Fatalf("parseFollowBidAmount(%q): %v", raw, err)
	}
	return amount
}

func signDemoToken(t *testing.T, secret string, userID int64) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": "demo",
		"role":     0,
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign demo token: %v", err)
	}
	return token
}
