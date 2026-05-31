package domain

import (
	"errors"
	"testing"
)

func TestOrderApplyFill(t *testing.T) {
	tests := []struct {
		name          string
		order         Order
		fillQtyUnits  int64
		wantStatus    OrderStatus
		wantRemaining int64
		wantErr       error
	}{
		{
			name: "partial fill keeps order pending",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 100,
				Status:                 OrderStatusPending,
			},
			fillQtyUnits:  40,
			wantStatus:    OrderStatusPending,
			wantRemaining: 60,
		},
		{
			name: "final fill marks order filled",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 40,
				Status:                 OrderStatusPending,
			},
			fillQtyUnits:  40,
			wantStatus:    OrderStatusFilled,
			wantRemaining: 0,
		},
		{
			name: "rejects zero fill",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 100,
				Status:                 OrderStatusPending,
			},
			fillQtyUnits:  0,
			wantStatus:    OrderStatusPending,
			wantRemaining: 100,
			wantErr:       ErrInvalidFillQuantity,
		},
		{
			name: "rejects overfill",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 30,
				Status:                 OrderStatusPending,
			},
			fillQtyUnits:  31,
			wantStatus:    OrderStatusPending,
			wantRemaining: 30,
			wantErr:       ErrFillQuantityExceedsRemaining,
		},
		{
			name: "rejects fill on canceled order",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 100,
				Status:                 OrderStatusCanceled,
			},
			fillQtyUnits:  10,
			wantStatus:    OrderStatusCanceled,
			wantRemaining: 100,
			wantErr:       ErrOrderTerminal,
		},
		{
			name: "rejects fill on filled order",
			order: Order{
				QuantityUnits:          100,
				RemainingQuantityUnits: 0,
				Status:                 OrderStatusFilled,
			},
			fillQtyUnits:  10,
			wantStatus:    OrderStatusFilled,
			wantRemaining: 0,
			wantErr:       ErrOrderTerminal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.order

			err := order.ApplyFill(tt.fillQtyUnits)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ApplyFill() error = %v, want %v", err, tt.wantErr)
			}
			if order.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", order.Status, tt.wantStatus)
			}
			if order.RemainingQuantityUnits != tt.wantRemaining {
				t.Fatalf("remaining quantity = %d, want %d", order.RemainingQuantityUnits, tt.wantRemaining)
			}
		})
	}
}

func TestOrderCancel(t *testing.T) {
	tests := []struct {
		name       string
		order      Order
		wantStatus OrderStatus
		wantErr    error
	}{
		{
			name: "cancels pending order",
			order: Order{
				Status: OrderStatusPending,
			},
			wantStatus: OrderStatusCanceled,
		},
		{
			name: "rejects cancel on filled order",
			order: Order{
				Status: OrderStatusFilled,
			},
			wantStatus: OrderStatusFilled,
			wantErr:    ErrOrderTerminal,
		},
		{
			name: "rejects cancel on canceled order",
			order: Order{
				Status: OrderStatusCanceled,
			},
			wantStatus: OrderStatusCanceled,
			wantErr:    ErrOrderTerminal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.order

			err := order.Cancel()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Cancel() error = %v, want %v", err, tt.wantErr)
			}
			if order.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", order.Status, tt.wantStatus)
			}
		})
	}
}
