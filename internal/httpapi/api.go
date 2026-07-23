package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fvlgnn/turnino/internal/credential"
	"github.com/fvlgnn/turnino/internal/domain"
)

type Store interface {
	AdminConfigured(context.Context) (bool, error)
	ValidateAdminToken(context.Context, []byte) (bool, error)
	CreateQueue(context.Context, domain.CreateQueueParams) (domain.Queue, error)
	CreateCounter(context.Context, string, []byte) (domain.Counter, error)
	RevokeCounter(context.Context, int64) error
	RotateCounterToken(context.Context, int64, []byte) error
	CallNext(context.Context, int64, []byte) (domain.Call, error)
}

type API struct {
	store Store
}

func New(store Store) http.Handler {
	api := &API{store: store}
	mux := http.NewServeMux()
	mux.Handle("POST /admin/queues", api.adminOnly(http.HandlerFunc(api.createQueue)))
	mux.Handle("POST /admin/counters", api.adminOnly(http.HandlerFunc(api.createCounter)))
	mux.Handle("DELETE /admin/counters/{counterID}", api.adminOnly(http.HandlerFunc(api.revokeCounter)))
	mux.Handle("POST /admin/counters/{counterID}/token", api.adminOnly(http.HandlerFunc(api.rotateCounterToken)))
	mux.HandleFunc("POST /queues/{queueID}/calls/next", api.callNext)
	return mux
}

func (a *API) createQueue(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Prefix   string `json:"prefix"`
		Digits   int    `json:"digits"`
		MinValue int    `json:"min_value"`
		MaxValue int    `json:"max_value"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	queue, err := a.store.CreateQueue(r.Context(), domain.CreateQueueParams(input))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, queueResponse(queue))
}

func (a *API) createCounter(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	token, tokenHash, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate counter token")
		return
	}
	counter, err := a.store.CreateCounter(r.Context(), input.Name, tokenHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	response := counterResponse(counter)
	response.Token = token
	writeJSON(w, http.StatusCreated, response)
}

func (a *API) revokeCounter(w http.ResponseWriter, r *http.Request) {
	counterID, err := parseID(r.PathValue("counterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid counter id")
		return
	}
	if err := a.store.RevokeCounter(r.Context(), counterID); errors.Is(err, domain.ErrCounterNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke counter")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) rotateCounterToken(w http.ResponseWriter, r *http.Request) {
	counterID, err := parseID(r.PathValue("counterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid counter id")
		return
	}
	token, tokenHash, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate counter token")
		return
	}
	if err := a.store.RotateCounterToken(r.Context(), counterID, tokenHash); errors.Is(err, domain.ErrCounterNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not rotate counter token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"counter_id": counterID, "token": token})
}

func (a *API) callNext(w http.ResponseWriter, r *http.Request) {
	queueID, err := parseID(r.PathValue("queueID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid queue id")
		return
	}
	token, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	tokenHash := credential.Hash(token)
	call, err := a.store.CallNext(r.Context(), queueID, tokenHash[:])
	if errors.Is(err, domain.ErrQueueNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, domain.ErrCounterUnavailable) {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not call next number")
		return
	}
	writeJSON(w, http.StatusCreated, callResponse(call))
}

func (a *API) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configured, err := a.store.AdminConfigured(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check administrator configuration")
			return
		}
		if !configured {
			writeError(w, http.StatusServiceUnavailable, "administrator is not configured")
			return
		}
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		tokenHash := credential.Hash(token)
		valid, err := a.store.ValidateAdminToken(r.Context(), tokenHash[:])
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not validate administrator token")
			return
		}
		if !valid {
			writeError(w, http.StatusUnauthorized, "invalid administrator token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}

func newToken() (string, []byte, error) {
	token, err := credential.Generate()
	if err != nil {
		return "", nil, err
	}
	tokenHash := credential.Hash(token)
	return token, tokenHash[:], nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

type queueJSON struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Prefix       string `json:"prefix"`
	Digits       int    `json:"digits"`
	MinValue     int    `json:"min_value"`
	MaxValue     int    `json:"max_value"`
	CurrentValue int    `json:"current_value"`
	CreatedAt    string `json:"created_at"`
}

func queueResponse(queue domain.Queue) queueJSON {
	return queueJSON{queue.ID, queue.Name, queue.Prefix, queue.Digits, queue.MinValue, queue.MaxValue, queue.CurrentValue, queue.CreatedAt.Format("2006-01-02T15:04:05Z")}
}

type counterJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	Token     string `json:"token,omitempty"`
}

func counterResponse(counter domain.Counter) counterJSON {
	return counterJSON{ID: counter.ID, Name: counter.Name, Enabled: counter.Enabled, CreatedAt: counter.CreatedAt.Format("2006-01-02T15:04:05Z")}
}

type callJSON struct {
	ID            int64           `json:"id"`
	QueueID       int64           `json:"queue_id"`
	CounterID     int64           `json:"counter_id"`
	Number        string          `json:"number"`
	SequenceValue int             `json:"sequence_value"`
	Kind          domain.CallKind `json:"kind"`
	CalledAt      string          `json:"called_at"`
}

func callResponse(call domain.Call) callJSON {
	return callJSON{call.ID, call.QueueID, call.CounterID, call.Number, call.SequenceValue, call.Kind, call.CalledAt.Format("2006-01-02T15:04:05Z")}
}
