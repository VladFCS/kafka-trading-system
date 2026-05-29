package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	orderv1 "github.com/vladfc/kafka-trading-system/gen/order/v1"
	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
	"github.com/vladfc/kafka-trading-system/internal/order-service/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCHandler struct {
	orderv1.UnimplementedOrderServiceServer
	service *service.OrderService
	logger  *slog.Logger
}

func NewGRPCHandler(service *service.OrderService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (h *GRPCHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	order, err := h.service.CreateOrder(ctx, domain.Order{
		CustomerID:     req.GetCustomerId(),
		Symbol:         req.GetSymbol(),
		Side:           mapProtoSideToDomain(req.GetSide()),
		PriceCents:     req.GetPriceCents(),
		QuantityUnits:  req.GetQuantityUnits(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOrder) {
			return nil, status.Error(codes.InvalidArgument, "invalid order")
		}
		return nil, status.Error(codes.Internal, "create order")
	}

	return &orderv1.CreateOrderResponse{
		Order: mapDomainOrderToProto(order),
	}, nil
}

func mapDomainOrderToProto(order domain.Order) *orderv1.Order {
	var idempotencyKey *string
	if order.IdempotencyKey != "" {
		idempotencyKey = &order.IdempotencyKey
	}

	return &orderv1.Order{
		OrderId:                order.OrderID,
		CustomerId:             order.CustomerID,
		Symbol:                 order.Symbol,
		Side:                   mapDomainSideToProto(order.Side),
		PriceCents:             order.PriceCents,
		QuantityUnits:          order.QuantityUnits,
		RemainingQuantityUnits: order.RemainingQuantityUnits,
		Status:                 mapDomainStatusToProto(order.Status),
		IdempotencyKey:         idempotencyKey,
		CanceledAt:             timeToProto(order.CanceledAt),
		CreatedAt:              timestamppb.New(order.CreatedAt),
		UpdatedAt:              timestamppb.New(order.UpdatedAt),
	}
}

func mapProtoSideToDomain(side orderv1.OrderSide) domain.OrderSide {
	switch side {
	case orderv1.OrderSide_ORDER_SIDE_BUY:
		return domain.OrderSideBuy
	case orderv1.OrderSide_ORDER_SIDE_SELL:
		return domain.OrderSideSell
	default:
		return ""
	}
}

func mapDomainSideToProto(side domain.OrderSide) orderv1.OrderSide {
	switch side {
	case domain.OrderSideBuy:
		return orderv1.OrderSide_ORDER_SIDE_BUY
	case domain.OrderSideSell:
		return orderv1.OrderSide_ORDER_SIDE_SELL
	default:
		return orderv1.OrderSide_ORDER_SIDE_UNSPECIFIED
	}
}

func mapDomainStatusToProto(status domain.OrderStatus) orderv1.OrderStatus {
	switch status {
	case domain.OrderStatusPending:
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case domain.OrderStatusFilled:
		return orderv1.OrderStatus_ORDER_STATUS_FILLED
	case domain.OrderStatusCanceled:
		return orderv1.OrderStatus_ORDER_STATUS_CANCELED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func timeToProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}
