package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"gRPC-USDT/api/proto"
	"gRPC-USDT/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockRateStorage struct {
	mock.Mock
}

func (m *MockRateStorage) SaveRate(ask, bid, askAmount, bidAmount float64, ts time.Time) error {
	args := m.Called(ask, bid, askAmount, bidAmount, ts)
	return args.Error(0)
}

func TestRateService_GetRateFromExchange(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{BinanceAPIURL: "http://test-api"}

	tests := []struct {
		name         string
		prepareMocks func(*MockHTTPClient, *MockRateStorage)
		wantErr      bool
		errContains  string
	}{
		{
			name: "successful rate fetch",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				respBody := `{"asks":[["50000.0","1.5"]],"bids":[["49900.0","2.0"]]}`
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(respBody)),
				}
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil).Once()
				storageMock.On("SaveRate", 50000.0, 49900.0, 2.0, 1.5, mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "HTTP request error",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).
					Return(nil, errors.New("network error")).Once()
			},
			wantErr:     true,
			errContains: "fetch rates failed",
		},
		{
			name: "invalid HTTP status",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				response := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(strings.NewReader("")),
				}
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil).Once()
			},
			wantErr:     true,
			errContains: "500 Internal Server Error",
		},
		{
			name: "empty order book",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				respBody := `{"asks":[],"bids":[]}`
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(respBody)),
				}
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil).Once()
			},
			wantErr:     true,
			errContains: "empty response from binance",
		},
		{
			name: "invalid JSON response",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{invalid json}")),
				}
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil).Once()
			},
			wantErr:     true,
			errContains: "decode response failed",
		},
		{
			name: "storage save error",
			prepareMocks: func(httpMock *MockHTTPClient, storageMock *MockRateStorage) {
				respBody := `{"asks":[["50000.0","1.5"]],"bids":[["49900.0","2.0"]]}`
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(respBody)),
				}
				httpMock.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil).Once()
				storageMock.On("SaveRate", 50000.0, 49900.0, 2.0, 1.5, mock.AnythingOfType("time.Time")).
					Return(errors.New("save error")).Once()
			},
			wantErr:     true,
			errContains: "save rate failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpMock := &MockHTTPClient{}
			storageMock := &MockRateStorage{}
			tt.prepareMocks(httpMock, storageMock)

			svc := NewRateService(storageMock, logger, cfg, httpMock)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			resp, err := svc.GetRateFromExchange(ctx, &proto.GetRateFromExchangeRequest{})

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.InDelta(t, 50000.0, float64(resp.Ask), 0.001)
				assert.InDelta(t, 49900.0, float64(resp.Bid), 0.001)
				assert.InDelta(t, 2.0, float64(resp.AskAmount), 0.001)
				assert.InDelta(t, 1.5, float64(resp.BidAmount), 0.001)
			}

			httpMock.AssertExpectations(t)
			storageMock.AssertExpectations(t)
		})
	}
}

func TestProcessOrder(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantPrice float64
		wantVol   float64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid order",
			input:     []string{"50000.0", "1.5"},
			wantPrice: 50000.0,
			wantVol:   1.5,
			wantErr:   false,
		},
		{
			name:    "invalid price format",
			input:   []string{"abc", "1.5"},
			wantErr: true,
			errMsg:  "price parsing error",
		},
		{
			name:    "invalid volume format",
			input:   []string{"50000.0", "abc"},
			wantErr: true,
			errMsg:  "volume parsing error",
		},
		{
			name:    "short array",
			input:   []string{"50000.0"},
			wantErr: true,
			errMsg:  "invalid order format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, vol, err := processOrder(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPrice, price)
				assert.Equal(t, tt.wantVol, vol)
			}
		})
	}
}
