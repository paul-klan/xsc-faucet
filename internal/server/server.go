package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LK4D4/trylock"
	"github.com/chainflag/eth-faucet/internal/chain"
	"github.com/chainflag/eth-faucet/web"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

const (
	AddressKey = "address"
	SymbolKey  = "symbol"
)

type Server struct {
	tx     chain.TxBuilder
	tokens map[string]*chain.TxTokenBuild
	mutex  trylock.Mutex
	cfg    *Config
	queue  chan string
}

func NewServer(builder chain.TxBuilder, tokens map[string]*chain.TxTokenBuild, cfg *Config) *Server {
	return &Server{
		tx:     builder,
		cfg:    cfg,
		tokens: tokens,
		queue:  make(chan string, cfg.queueCap),
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/", http.FileServer(web.Dist()))
	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
	router.Handle("/api/claim", negroni.New(limiter, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/info", s.handleInfo())

	return router
}

func (s *Server) Run() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			s.consumeQueue()
		}
	}()

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())
	log.Infof("Starting http server %d", s.cfg.httpPort)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(s.cfg.httpPort), n))
}

func (s *Server) consumeQueue() {
	if len(s.queue) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for len(s.queue) != 0 {
		q := <-s.queue
		var txHash common.Hash
		var txErr error

		lists := strings.Split(q, ":")
		address := lists[0]
		var symbol string
		if len(lists) > 1 {
			symbol = lists[1]
		}

		log.Infof("address %s symbol %s", address, symbol)
		if symbol == "" {
			txHash, txErr = s.tx.Transfer(context.Background(), address, chain.EtherToWei(int64(s.cfg.payout)))
		} else {
			log.Infof("tx address %s symbol %s", address, symbol)
			token := s.tokens[strings.ToLower(symbol)]
			log.Infof("token %#v symbol %s", token, symbol)
			txHash, txErr = token.Transfer(context.Background(), address, chain.EtherTokenAmount(int64(s.cfg.payout)))
		}

		if txErr != nil {
			log.WithError(txErr).Error("Failed to handle transaction in the queue")
		} else {
			log.WithFields(log.Fields{
				"txHash":  txHash,
				"address": address,
			}).Info("Consume from queue successfully")
		}
	}
}

func (s *Server) handleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		address := r.PostFormValue(AddressKey)
		symbol := r.PostFormValue(SymbolKey)

		log.Infof("address %s symbol %s", address, symbol)
		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- address + ":" + symbol:
				log.WithFields(log.Fields{
					"address": address,
				}).Info("Added to queue successfully")
				fmt.Fprintf(w, "Added %s to the queue", address)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var txHash common.Hash
		var txErr error
		if symbol == "" {
			txHash, txErr = s.tx.Transfer(ctx, address, chain.EtherToWei(int64(s.cfg.payout)))
		} else {
			log.Infof("tx address %s symbol %s", address, symbol)
			token := s.tokens[strings.ToLower(symbol)]
			log.Infof("token %#v symbol %s", token, symbol)
			txHash, txErr = token.Transfer(ctx, address, chain.EtherTokenAmount(int64(s.cfg.payout)))
		}

		s.mutex.Unlock()
		if txErr != nil {
			log.WithError(txErr).Error("Failed to send transaction")
			http.Error(w, txErr.Error(), http.StatusInternalServerError)
			return
		}

		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": address,
		}).Info("Funded directly successfully")
		fmt.Fprintf(w, "Txhash: %s", txHash)
	}
}

func (s *Server) handleInfo() http.HandlerFunc {
	type info struct {
		Account string `json:"account"`
		Network string `json:"network"`
		Payout  string `json:"payout"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info{
			Account: s.tx.Sender().String(),
			Network: s.cfg.network,
			Payout:  strconv.Itoa(s.cfg.payout),
		})
	}
}
