package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gRPC-USDT/api/proto"
	"gRPC-USDT/internal/config"
	"gRPC-USDT/internal/models"
	"go.uber.org/zap"
)

// HTTPClient интерфейс для HTTP клиента
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RateStorage интерфейс для работы с хранилищем курсов
type RateStorage interface {
	SaveRate(ask, bid, askAmount, bidAmount float64, ts time.Time) error
}

// DefaultHTTPClient реализация HTTPClient по умолчанию
type DefaultHTTPClient struct{}

func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// RateService сервис работы с курсами
type RateService struct {
	proto.UnimplementedRateServiceServer
	storage    RateStorage
	logger     *zap.Logger
	cfg        *config.Config
	httpClient HTTPClient
}

// NewRateService создает новый экземпляр RateService
func NewRateService(
	storage RateStorage,
	logger *zap.Logger,
	cfg *config.Config,
	httpClient HTTPClient,
) *RateService {
	if httpClient == nil {
		httpClient = &DefaultHTTPClient{}
	}
	return &RateService{
		storage:    storage,
		logger:     logger,
		cfg:        cfg,
		httpClient: httpClient,
	}
}

// GetRateFromExchange получает курс от биржи и сохраняет его
func (s *RateService) GetRateFromExchange(
	ctx context.Context,
	req *proto.GetRateFromExchangeRequest,
) (*proto.GetRateFromExchangeResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", s.cfg.BinanceAPIURL, nil)
	if err != nil {
		s.logger.Error("Error creating request", zap.Error(err))
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.Error("Error fetching rates", zap.Error(err))
		return nil, fmt.Errorf("fetch rates failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance API returned status: %s", resp.Status)
	}

	var depthResponse models.BinanceDepthResponse
	if err := json.NewDecoder(resp.Body).Decode(&depthResponse); err != nil {
		s.logger.Error("Error decoding response", zap.Error(err))
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if len(depthResponse.Asks) == 0 || len(depthResponse.Bids) == 0 {
		return nil, fmt.Errorf("empty response from binance")
	}

	bestAsk, bidVolume, err := processOrder(depthResponse.Asks[0])
	if err != nil {
		return nil, fmt.Errorf("ask processing failed: %w", err)
	}

	bestBid, askVolume, err := processOrder(depthResponse.Bids[0])
	if err != nil {
		return nil, fmt.Errorf("bid processing failed: %w", err)
	}

	timestamp := time.Now()
	if err := s.storage.SaveRate(bestAsk, bestBid, askVolume, bidVolume, timestamp); err != nil {
		s.logger.Error("Error saving rate", zap.Error(err))
		return nil, fmt.Errorf("save rate failed: %w", err)
	}
	s.logger.Info("Rate saved successfully")

	return &proto.GetRateFromExchangeResponse{
		Success:   true,
		Ask:       float32(bestAsk),
		Bid:       float32(bestBid),
		AskAmount: float32(askVolume),
		BidAmount: float32(bidVolume),
		Timestamp: timestamp.Format(time.RFC3339),
	}, nil
}

func processOrder(order []string) (price, volume float64, err error) {
	if len(order) < 2 {
		return 0, 0, fmt.Errorf("invalid order format")
	}

	price, err = strconv.ParseFloat(order[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("price parsing error: %w", err)
	}

	volume, err = strconv.ParseFloat(order[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("volume parsing error: %w", err)
	}

	return price, volume, nil
}
