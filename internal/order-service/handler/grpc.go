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
		return nil, mapOrderError(err)
	}

	return &orderv1.CreateOrderResponse{
		Order: mapDomainOrderToProto(order),
	}, nil
}

func (h *GRPCHandler) GetOrderByID(ctx context.Context, req *orderv1.GetOrderByIDRequest) (*orderv1.GetOrderByIDResponse, error) {
	order, err := h.service.GetOrderByID(ctx, req.GetOrderId())
	if err != nil {
		return nil, mapOrderError(err)
	}

	return &orderv1.GetOrderByIDResponse{
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

func mapOrderError(err error) error {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMissingOrderID),
		errors.Is(err, domain.ErrMissingCustomerID),
		errors.Is(err, domain.ErrMissingSymbol),
		errors.Is(err, domain.ErrInvalidOrderSide),
		errors.Is(err, domain.ErrInvalidPriceCents),
		errors.Is(err, domain.ErrInvalidQuantityUnits),
		errors.Is(err, domain.ErrInvalidRemainingQuantityUnits),
		errors.Is(err, domain.ErrRemainingQuantityExceedsQuantity),
		errors.Is(err, domain.ErrMissingOrderStatus),
		errors.Is(err, domain.ErrInvalidOrderStatus),
		errors.Is(err, domain.ErrCanceledOrderOnCreate),
		errors.Is(err, domain.ErrInvalidOrder):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "order service error")
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
