package server

type Config struct {
	network    string
	httpPort   int
	interval   int
	payout     int
	proxyCount int
	queueCap   int
}

func NewConfig(network string, httpPort, interval, payout, proxyCount, queueCap int) *Config {
	return &Config{
		network:    network,
		httpPort:   httpPort,
		interval:   interval,
		payout:     payout,
		proxyCount: proxyCount,
		queueCap:   queueCap,
	}
}

type Erc20Token struct {
	ContractAddress string `json:"contract_address"`
	Decimal         int    `json:"decimal,omitempty"`
	Symbol          string `json:"symbol"`
}

type Erc20Tokens struct {
	Tokens []Erc20Tokens `json:"tokens,omitempty"`
}
