package domain

// ServiceType mirrors products.service_type.
type ServiceType string

const (
	ServiceCard      ServiceType = "card"
	ServiceTopupData ServiceType = "topup_data"
	ServiceTopup     ServiceType = "topup"
)

// RoutingMode mirrors products.routing_mode.
type RoutingMode string

const (
	RoutingSKU      RoutingMode = "sku"
	RoutingProvider RoutingMode = "provider"
)

// DataSource mirrors agent_settings.data_source.
type DataSource string

const (
	DataSourceMock       DataSource = "mock"
	DataSourceProduction DataSource = "production"
)

// Product catalog row.
type Product struct {
	ID          uint
	ProductCode string
	Label       string
	ServiceType ServiceType
	RoutingMode RoutingMode
	Enabled     bool
}

// Provider linked to a product.
type Provider struct {
	ProviderCode string
	Label        string
	Enabled      bool
}

// SKU under a product.
type SKU struct {
	SKUCode           string
	Label             string
	Enabled           bool
}

// RoutingPct row for a product/sku/provider.
type RoutingPct struct {
	ProductCode  string
	SKUCode      string
	ProviderCode string
	BaselinePct  float64
	TrafficPct   float64
}
