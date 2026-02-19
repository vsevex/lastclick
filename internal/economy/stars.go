package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/lastclick/lastclick/internal/store"
)

// StarsService handles Telegram Stars payment flow.
type StarsService struct {
	botToken string
	players  *store.PlayerStore
	txs      *store.TransactionStore
	logger   *slog.Logger
}

func NewStarsService(botToken string, players *store.PlayerStore, txs *store.TransactionStore, logger *slog.Logger) *StarsService {
	return &StarsService{
		botToken: botToken,
		players:  players,
		txs:      txs,
		logger:   logger,
	}
}

// CreateInvoiceLink generates a Telegram Stars invoice link for purchasing life shares.
func (s *StarsService) CreateInvoiceLink(ctx context.Context, title string, description string, amount int) (string, error) {
	payload := map[string]any{
		"title":       title,
		"description": description,
		"payload":     fmt.Sprintf("stars_purchase_%d", amount),
		"currency":    "XTR",
		"prices":      []map[string]any{{"label": title, "amount": amount}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("https://api.telegram.org/bot%s/createInvoiceLink", s.botToken),
		jsonReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("telegram api: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("telegram api returned not ok")
	}
	return result.Result, nil
}

// HandlePreCheckout answers a pre_checkout_query (must respond within 10 seconds).
func (s *StarsService) HandlePreCheckout(ctx context.Context, queryID string, ok bool) error {
	payload := map[string]any{
		"pre_checkout_query_id": queryID,
		"ok":                    ok,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("https://api.telegram.org/bot%s/answerPreCheckoutQuery", s.botToken),
		jsonReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreditStars adds Stars to a player's balance after successful payment.
func (s *StarsService) CreditStars(ctx context.Context, playerID int64, amount int64) error {
	if err := s.players.UpdateBalance(ctx, playerID, amount, 0); err != nil {
		return err
	}
	return s.txs.Record(ctx, playerID, store.TxEntry, amount, nil)
}

// DebitStars deducts Stars for a pulse or entry fee.
func (s *StarsService) DebitStars(ctx context.Context, playerID int64, amount int64, roomID *string) error {
	player, err := s.players.Get(ctx, playerID)
	if err != nil {
		return err
	}
	if player == nil || player.StarsBalance < amount {
		return fmt.Errorf("insufficient stars balance")
	}
	if err := s.players.UpdateBalance(ctx, playerID, -amount, 0); err != nil {
		return err
	}
	return s.txs.Record(ctx, playerID, store.TxPulse, -amount, roomID)
}
