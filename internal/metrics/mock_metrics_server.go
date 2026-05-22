//go:build !prod

package metrics

import (
	"context"
	"fmt"

	"github.com/andrewbytecoder/mgap/api"
	"github.com/andrewbytecoder/mgap/internal/logging"
)

// MockMetricsServer returns mock data instead of real data to save development time
type MockMetricsServer struct {
	api.UnimplementedMockMetricsServer

	mockResources *MockAssets
}

func NewMockMetricsServer() *MockMetricsServer {
	return &MockMetricsServer{
		mockResources: newMockAssets(),
	}
}

func (m *MockMetricsServer) HeapMetrics(_ context.Context, req *api.GoMetricsRequest) (*api.GoMetricsResponse, error) {
	logging.Sugar.Debug("HeapMetrics req:", req)
	return m.dispatch(req, MetricsTypeHeap)
}

func (m *MockMetricsServer) CPUMetrics(_ context.Context, req *api.GoMetricsRequest) (*api.GoMetricsResponse, error) {
	logging.Sugar.Debug("CPUMetrics req:", req)
	return m.dispatch(req, MetricsTypeCPU)
}

func (m *MockMetricsServer) AllocsMetrics(_ context.Context, req *api.GoMetricsRequest) (*api.GoMetricsResponse, error) {
	logging.Sugar.Debug("AllocsMetrics req:", req)
	return m.dispatch(req, MetricsTypeAllocs)
}

func (m *MockMetricsServer) GoroutineMetrics(_ context.Context, req *api.GoMetricsRequest) (*api.GoMetricsResponse, error) {
	logging.Sugar.Debug("GoroutineMetrics req:", req)
	return m.dispatch(req, MetricsTypeGoroutine)
}

func (m *MockMetricsServer) GenericMetrics(req *api.GoMetricsRequest, mt MetricsType) (*api.GoMetricsResponse, error) {
	switch mt {
	case MetricsTypeHeap, MetricsTypeCPU, MetricsTypeAllocs, MetricsTypeGoroutine:
		return m.dispatch(req, mt)
	default:
		return nil, fmt.Errorf("mock data is not available for metric %s", mt)
	}
}

func (m *MockMetricsServer) dispatch(req *api.GoMetricsRequest, mt MetricsType) (*api.GoMetricsResponse, error) {
	_, err := MetricsURL(false, mt, req.Url, req.ProfileSeconds)
	if err != nil {
		return nil, err
	}

	mtr, err := m.mockResources.GetMetrics(mt)
	if err != nil {
		logging.Sugar.Error(err)
		return nil, err
	}
	resp := toResp(mtr)

	return resp, nil
}
