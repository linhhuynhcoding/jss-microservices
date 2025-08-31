package service

import (
	"context"
	"fmt"

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
	customerId int32,
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
	for _, voucher := range voucherIds {
		for _, custVoucher := range custVouchers {
			if custVoucher.Code != voucher {
				logger.Error("invalid voucher code", zap.String("voucher", voucher))
				return nil, fmt.Errorf("invalid voucher code: %s", voucher)
			}
			voucherDb, err := s.queries.GetVoucherByCode(ctx, voucher)
			if err != nil {
				logger.Error("failed to get voucher", zap.String("voucher", voucher))
				return nil, fmt.Errorf("failed to get voucher: %w", err)
			}
			vouchers = append(vouchers, voucherDb)
		}
	}

	return vouchers, nil
}
