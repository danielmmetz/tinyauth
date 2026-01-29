package service

import (
	"context"
	"net"
	"strings"
	"time"
	"tinyauth/internal/model"
	"tinyauth/internal/utils"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type IPBypassService struct {
	database *gorm.DB
	docker   *DockerService
}

func NewIPBypassService(database *gorm.DB, docker *DockerService) *IPBypassService {
	return &IPBypassService{database: database, docker: docker}
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

func (s *IPBypassService) List(ctx context.Context, username string, isAdmin bool) ([]model.IPBypass, error) {
	if s == nil {
		return nil, nil
	}

	var bypasses []model.IPBypass
	query := s.database.WithContext(ctx)

	// Non-admin users can only see their own bypasses
	if !isAdmin {
		query = query.Where("created_by = ?", username)
	}

	err := query.Find(&bypasses).Error
	if err != nil {
		log.Error().Err(err).Msg("Failed to list IP bypasses")
		return nil, err
	}

	return bypasses, nil
}

func (s *IPBypassService) Create(ctx context.Context, bypass *model.IPBypass) error {
	if s == nil {
		return nil
	}

	// Validate CIDR/IP format
	cidrToValidate := strings.ReplaceAll(bypass.CIDR, "-", "/")
	if strings.Contains(cidrToValidate, "/") {
		_, _, err := net.ParseCIDR(cidrToValidate)
		if err != nil {
			log.Warn().Str("cidr", bypass.CIDR).Err(err).Msg("Invalid CIDR format")
			return gorm.ErrInvalidData
		}
	} else {
		if net.ParseIP(cidrToValidate) == nil {
			log.Warn().Str("cidr", bypass.CIDR).Msg("Invalid IP address format")
			return gorm.ErrInvalidData
		}
	}

	bypass.CreatedAt = time.Now().Unix()

	err := s.database.WithContext(ctx).Create(bypass).Error
	if err != nil {
		log.Error().Err(err).Msg("Failed to create IP bypass")
		return err
	}

	return nil
}

func (s *IPBypassService) Delete(ctx context.Context, id int64, username string, isAdmin bool) error {
	if s == nil {
		return nil
	}

	// Non-admin users can only delete their own bypasses
	if !isAdmin {
		result := s.database.WithContext(ctx).
			Where("id = ? AND created_by = ?", id, username).
			Delete(&model.IPBypass{})

		if result.Error != nil {
			log.Error().Err(result.Error).Msg("Failed to delete IP bypass")
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	}

	// Admin can delete any bypass
	result := s.database.WithContext(ctx).Delete(&model.IPBypass{}, id)
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("Failed to delete IP bypass")
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *IPBypassService) GetDomains(ctx context.Context) ([]string, error) {
	if s == nil || s.docker == nil {
		return []string{"*"}, nil
	}

	return s.docker.GetDomains()
}
