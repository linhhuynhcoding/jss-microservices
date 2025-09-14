package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	utils "github.com/linhhuynhcoding/jss-microservices/jss-shared/utils/format"
	db "github.com/linhhuynhcoding/jss-microservices/loyalty/internal/repository"
	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/loyalty"
	"go.uber.org/zap"
)

func (s *Service) calculateDiscountAmount(
	vouchers []db.Voucher,
	totalProductAmount float64,
) ([]*api.CalculateDiscountAmountResponse_Voucher, float64) {
	totalDiscountAmount := float64(0)
	vouchersResp := make([]*api.CalculateDiscountAmountResponse_Voucher, 0)
	s.logger.Info("vouchers", zap.Any("vouchers", vouchers))

	for _, voucher := range vouchers {
		discountAmount := float64(0)
		switch voucher.DiscountType {
		case "PERCENTAGE":
			discountAmount = totalProductAmount * (utils.NumericToFloat64(voucher.DiscountValue) / 100)
		case "FIXED":
			discountAmount = utils.NumericToFloat64(voucher.DiscountValue)
		}

		v := api.CalculateDiscountAmountResponse_Voucher{
			Id:             voucher.ID,
			Code:           voucher.Code,
			Title:          voucher.Description.String,
			DiscountAmount: discountAmount,
		}
		vouchersResp = append(vouchersResp, &v)
		totalDiscountAmount += discountAmount
	}

	return vouchersResp, totalDiscountAmount
}

func (s *Service) validateVoucher(
	ctx context.Context,
	customerId string,
	voucherIds []string,
) ([]db.Voucher, error) {
	logger := s.logger.With(zap.Any("method", "ValidateVoucher"))

	vouchers := make([]db.Voucher, 0)
	custVouchers, err := s.queries.GetCustomerVouchers(ctx, db.GetCustomerVouchersParams{
		CustomerID: customerId,
		Limit:      1000000,
		Offset:     0,
	})
	if err != nil || len(custVouchers) == 0 {
		logger.Error("failed to get customer vouchers")
		return nil, fmt.Errorf("failed to get customer vouchers: %w", err)
	}
	for _, custVoucher := range custVouchers {
		if slices.Contains(voucherIds, custVoucher.Code) {
			if custVoucher.Status.String == "used" {
				logger.Error("voucher is used")
				return nil, fmt.Errorf("voucher is used")
			}
			if custVoucher.StartDate.Time.After(time.Now()) {
				logger.Error("voucher is not active")
				return nil, fmt.Errorf("voucher is not active")
			}
			if custVoucher.EndDate.Time.Before(time.Now()) {
				logger.Error("voucher is expired")
				return nil, fmt.Errorf("voucher is expired")
			}
			vouchers = append(vouchers, db.Voucher{
				ID:            custVoucher.ID,
				Code:          custVoucher.Code,
				Description:   custVoucher.Description,
				DiscountType:  custVoucher.DiscountType,
				DiscountValue: custVoucher.DiscountValue,
				StartDate:     custVoucher.StartDate,
				EndDate:       custVoucher.EndDate,
				UsageLimit:    custVoucher.UsageLimit,
				IsGlobal:      custVoucher.IsGlobal,	
			})
		}
	}
	return vouchers, nil
}
