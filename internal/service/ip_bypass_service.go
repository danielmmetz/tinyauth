package service

import (
	"context"
	"time"
	"tinyauth/internal/model"
	"tinyauth/internal/utils"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type IPBypassService struct {
	database *gorm.DB
}

func NewIPBypassService(database *gorm.DB) *IPBypassService {
	return &IPBypassService{database: database}
}

func (s *IPBypassService) IsBypassed(ctx context.Context, domain, clientIP string) bool {
	if s == nil {
		return false
	}

	var bypasses []model.IPBypass
	err := s.database.WithContext(ctx).
		Where(
			"(domain = ? OR domain = '*') AND expires_at > ?",
			domain,
			time.Now().Unix(),
		).
		Find(&bypasses).Error

	if err != nil {
		log.Error().Err(err).Msg("Failed to query IP bypasses")
		return false
	}

	for _, bypass := range bypasses {
		match, err := utils.FilterIP(bypass.CIDR, clientIP)
		if err != nil {
			log.Warn().Err(err).Str("cidr", bypass.CIDR).Msg("Invalid CIDR in IP bypass table")
			continue
		}
		if match {
			log.Debug().
				Str("ip", clientIP).
				Str("cidr", bypass.CIDR).
				Str("domain", bypass.Domain).
				Msg("IP matched dynamic bypass rule, allowing access")
			return true
		}
	}

	return false
}

func (s *IPBypassService) Cleanup(ctx context.Context) error {
	if s == nil {
		return nil
	}
	result := s.database.WithContext(ctx).
		Where("expires_at < ?", time.Now().Unix()).
		Delete(&model.IPBypass{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Debug().Int64("count", result.RowsAffected).Msg("Cleaned up expired IP bypasses")
	}

	return nil
}
