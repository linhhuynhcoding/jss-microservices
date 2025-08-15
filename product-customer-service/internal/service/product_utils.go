package service

import (
	"time"

	db "github.com/linhhuynhcoding/jss-microservices/product/internal/repository"
	utils "github.com/linhhuynhcoding/jss-microservices/product/internal/utils/numeric"
	api "github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
)

func (s *Service) productToProto(p db.Product) *api.Product {
	return &api.Product{
		Id:              p.ID,
		Name:            p.Name,
		Code:            p.Code,
		CategoryId:      p.CategoryID,
		Weight:          utils.NumericToFloat64(p.Weight),
		GoldPriceAtTime: utils.NumericToFloat64(p.GoldPriceAtTime),
		LaborCost:       utils.NumericToFloat64(p.LaborCost),
		StoneCost:       utils.NumericToFloat64(p.StoneCost),
		MarkupRate:      utils.NumericToFloat64(p.MarkupRate),
		SellingPrice:    utils.NumericToFloat64(p.SellingPrice),
		WarrantyPeriod:  p.WarrantyPeriod.Int32,
		Image:           p.Image,
		CreatedAt:       p.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Time.Format(time.RFC3339),
	}
}
