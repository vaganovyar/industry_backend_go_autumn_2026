package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
	ErrInvalidBody  = errors.New("invalid body")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		mu:    sync.RWMutex{},
		clock: clock,
		seq:   0,
		tasks: make(map[string]Task),
	}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	newTask := Task{
		ID:        strconv.FormatUint(r.seq, 10),
		Title:     trimmedTitle,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.seq++
	r.tasks[newTask.ID] = newTask

	return newTask, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.tasks[id]
	if !ok {
		return Task{}, false
	}

	return task, ok
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	tasks := make([]Task, 0)
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, task := range r.tasks {
		if task.Done == done {
			tasks = append(tasks, task)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		left := tasks[i]
		right := tasks[j]
		if left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.ID < right.ID
		}

		return left.UpdatedAt.After(right.UpdatedAt)
	})

	return tasks
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()
		r.tasks[id] = task
	}

	return task, nil
}

var (
	tasksRegexp    = regexp.MustCompile(`^/tasks$`)
	taskByIDRegexp = regexp.MustCompile(`^/tasks/([a-zA-Z0-9]+)$`)
)

type CreateRequest struct {
	Title *string `json:"title"`
}

type PatchRequest struct {
	Done *bool `json:"done"`
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case tasksRegexp.MatchString(r.URL.Path) && r.Method == http.MethodPost:
		h.handleCreate(w, r)

	case taskByIDRegexp.MatchString(r.URL.Path) && r.Method == http.MethodGet:
		matches := taskByIDRegexp.FindStringSubmatch(r.URL.Path)
		id := matches[1]
		h.handleGet(w, r, id)

	case tasksRegexp.MatchString(r.URL.Path) && r.Method == http.MethodGet:
		h.handleList(w, r)

	case taskByIDRegexp.MatchString(r.URL.Path) && r.Method == http.MethodPatch:
		matches := taskByIDRegexp.FindStringSubmatch(r.URL.Path)
		id := matches[1]
		h.handlePatch(w, r, id)

	case tasksRegexp.MatchString(r.URL.Path) || taskByIDRegexp.MatchString(r.URL.Path):
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	default:
		http.NotFound(w, r)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body CreateRequest
	err := decodeStrictJSON(r.Body, &body)
	if err != nil ||
		body.Title == nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*body.Title)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTitle):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	writeJSON(w, http.StatusCreated, task)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	doneStr, exists := query["done"]
	done := false

	if exists {
		if len(doneStr) != 1 || (doneStr[0] != "true" && doneStr[0] != "false") {
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}

		var err error
		done, err = strconv.ParseBool(doneStr[0])
		if err != nil {
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body PatchRequest
	err := decodeStrictJSON(r.Body, &body)
	if err != nil ||
		body.Done == nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *body.Done)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return ErrInvalidBody
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidBody
	}

	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
