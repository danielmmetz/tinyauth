package service

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/steveiliop56/tinyauth/internal/repository"
	"github.com/steveiliop56/tinyauth/internal/utils"
	"github.com/steveiliop56/tinyauth/internal/utils/tlog"
)

var ErrInvalidCIDR = errors.New("invalid CIDR or IP address")

type IPBypassService struct {
	queries *repository.Queries
	docker  *DockerService
}

func NewIPBypassService(queries *repository.Queries, docker *DockerService) *IPBypassService {
	return &IPBypassService{queries: queries, docker: docker}
}

func (s *IPBypassService) IsBypassed(ctx context.Context, domain, clientIP string) bool {
	if s == nil {
		return false
	}

	bypasses, err := s.queries.ListActiveIPBypassesForDomain(ctx, repository.ListActiveIPBypassesForDomainParams{
		Domain:    domain,
		ExpiresAt: time.Now().Unix(),
	})
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to query IP bypasses")
		return false
	}

	for _, bypass := range bypasses {
		match, err := utils.FilterIP(bypass.Cidr, clientIP)
		if err != nil {
			tlog.App.Warn().Err(err).Str("cidr", bypass.Cidr).Msg("Invalid CIDR in IP bypass table")
			continue
		}
		if match {
			tlog.App.Debug().
				Str("ip", clientIP).
				Str("cidr", bypass.Cidr).
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
	rows, err := s.queries.DeleteExpiredIPBypasses(ctx, time.Now().Unix())
	if err != nil {
		return err
	}
	if rows > 0 {
		tlog.App.Debug().Int64("count", rows).Msg("Cleaned up expired IP bypasses")
	}
	return nil
}

func (s *IPBypassService) List(ctx context.Context, username string, isAdmin bool) ([]repository.IpBypass, error) {
	if s == nil {
		return nil, nil
	}

	if isAdmin {
		return s.queries.ListIPBypasses(ctx)
	}
	return s.queries.ListIPBypassesByCreator(ctx, username)
}

func (s *IPBypassService) Create(ctx context.Context, params repository.CreateIPBypassParams) (repository.IpBypass, error) {
	if s == nil {
		return repository.IpBypass{}, nil
	}

	cidrToValidate := strings.ReplaceAll(params.Cidr, "-", "/")
	if strings.Contains(cidrToValidate, "/") {
		if _, _, err := net.ParseCIDR(cidrToValidate); err != nil {
			tlog.App.Warn().Str("cidr", params.Cidr).Err(err).Msg("Invalid CIDR format")
			return repository.IpBypass{}, ErrInvalidCIDR
		}
	} else {
		if net.ParseIP(cidrToValidate) == nil {
			tlog.App.Warn().Str("cidr", params.Cidr).Msg("Invalid IP address format")
			return repository.IpBypass{}, ErrInvalidCIDR
		}
	}

	params.CreatedAt = time.Now().Unix()

	bypass, err := s.queries.CreateIPBypass(ctx, params)
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to create IP bypass")
		return repository.IpBypass{}, err
	}
	return bypass, nil
}

func (s *IPBypassService) Delete(ctx context.Context, id int64, username string, isAdmin bool) error {
	if s == nil {
		return nil
	}

	if isAdmin {
		if err := s.queries.DeleteIPBypass(ctx, id); err != nil {
			tlog.App.Error().Err(err).Msg("Failed to delete IP bypass")
			return err
		}
		return nil
	}

	rows, err := s.queries.DeleteIPBypassForCreator(ctx, repository.DeleteIPBypassForCreatorParams{
		ID:        id,
		CreatedBy: username,
	})
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to delete IP bypass")
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *IPBypassService) GetDomains(ctx context.Context) ([]string, error) {
	if s == nil || s.docker == nil {
		return []string{"*"}, nil
	}
	return s.docker.GetDomains()
}
