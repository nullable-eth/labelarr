package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/media"
	"github.com/nullable-eth/labelarr/internal/plex"
)

const eventLibraryNew = "library.new"

// PlexWebhookPayload matches the Plex webhook JSON structure.
// Plex sends this as the "payload" field in a multipart/form-data POST.
// Requires Plex Pass on the server.
type PlexWebhookPayload struct {
	Event   string `json:"event"`
	User    bool   `json:"user"`
	Owner   bool   `json:"owner"`
	Account struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"Account"`
	Server struct {
		Title string `json:"title"`
		UUID  string `json:"uuid"`
	} `json:"Server"`
	Player struct {
		Local bool   `json:"local"`
		Title string `json:"title"`
		UUID  string `json:"uuid"`
	} `json:"Player"`
	Metadata struct {
		LibrarySectionType  string `json:"librarySectionType"`
		LibrarySectionID    int    `json:"librarySectionID"`
		LibrarySectionTitle string `json:"librarySectionTitle"`
		RatingKey           string `json:"ratingKey"`
		Key                 string `json:"key"`
		Type                string `json:"type"`
		Title               string `json:"title"`
		Year                int    `json:"year"`
		AddedAt             int64  `json:"addedAt"`
	} `json:"Metadata"`
}

type libraryInfo struct {
	name      string
	mediaType media.MediaType
}

// pendingWork tracks accumulated rating keys during a debounce window.
type pendingWork struct {
	libraryName string
	mediaType   media.MediaType
	ratingKeys  map[string]struct{}
	timer       *time.Timer
	gen         uint64
}

// Scanner kicks off full or per-library scan cycles. Implemented by main.
type Scanner interface {
	RunAll()
	RunLibrary(libraryID, libraryName string, mediaType media.MediaType) error
}

type Server struct {
	config     *config.Config
	processor  *media.Processor
	scanner    Scanner
	httpServer *http.Server
	libraryMap map[string]libraryInfo
	pending    map[string]*pendingWork
	pendingMu  sync.Mutex
	scanMu     sync.Mutex
	scanning   bool
}

func NewServer(cfg *config.Config, proc *media.Processor, movieLibs, tvLibs []plex.Library, scanner Scanner) *Server {
	libMap := make(map[string]libraryInfo, len(movieLibs)+len(tvLibs))
	for _, lib := range movieLibs {
		libMap[lib.Key] = libraryInfo{name: lib.Title, mediaType: media.MediaTypeMovie}
	}
	for _, lib := range tvLibs {
		libMap[lib.Key] = libraryInfo{name: lib.Title, mediaType: media.MediaTypeTV}
	}

	return &Server{
		config:     cfg,
		processor:  proc,
		scanner:    scanner,
		libraryMap: libMap,
		pending:    make(map[string]*pendingWork),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/scan", s.handleScan)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	addr := fmt.Sprintf(":%d", s.config.WebhookPort)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind webhook port %d: %w", s.config.WebhookPort, err)
	}

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[WEBHOOK] server error: %v\n", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		fmt.Printf("[WEBHOOK] 400 ParseMultipartForm: content-type=%q content-length=%d err=%v\n",
			r.Header.Get("Content-Type"), r.ContentLength, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	payloadStr := r.FormValue("payload")
	if payloadStr == "" {
		formKeys := make([]string, 0, len(r.Form))
		for k := range r.Form {
			formKeys = append(formKeys, k)
		}
		fileKeys := []string{}
		if r.MultipartForm != nil {
			for k := range r.MultipartForm.File {
				fileKeys = append(fileKeys, k)
			}
		}
		fmt.Printf("[WEBHOOK] 400 MissingPayload: content-type=%q form_keys=%v file_keys=%v\n",
			r.Header.Get("Content-Type"), formKeys, fileKeys)
		http.Error(w, "Missing payload", http.StatusBadRequest)
		return
	}

	var payload PlexWebhookPayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		snippet := payloadStr
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		fmt.Printf("[WEBHOOK] 400 InvalidPayload: err=%v payload_snippet=%q\n", err, snippet)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if s.config.VerboseLogging {
		fmt.Printf("[WEBHOOK] received: event=%s library=%s section_type=%s media_type=%s title=%s\n",
			payload.Event,
			payload.Metadata.LibrarySectionTitle,
			payload.Metadata.LibrarySectionType,
			payload.Metadata.Type,
			payload.Metadata.Title)
	}

	if payload.Event != eventLibraryNew {
		w.WriteHeader(http.StatusOK)
		return
	}

	libraryID := strconv.Itoa(payload.Metadata.LibrarySectionID)
	mediaType := s.resolveMediaType(libraryID, payload.Metadata.LibrarySectionType)
	libraryName := payload.Metadata.LibrarySectionTitle
	if libraryName == "" {
		if info, ok := s.libraryMap[libraryID]; ok {
			libraryName = info.name
		}
	}

	if mediaType != media.MediaTypeUnknown {
		s.addPendingItem(libraryID, libraryName, mediaType, payload.Metadata.RatingKey)
	} else if s.config.VerboseLogging {
		fmt.Printf("[WEBHOOK] ignoring event for unknown library %s (ID: %s)\n", libraryName, libraryID)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.scanner == nil {
		http.Error(w, "Scanner unavailable", http.StatusServiceUnavailable)
		return
	}

	libParam := r.URL.Query().Get("library")

	var (
		targetID   string
		targetName string
		targetMT   media.MediaType
	)
	if libParam != "" {
		id, info, ok := s.findLibrary(libParam)
		if !ok {
			http.Error(w, "Library not found", http.StatusNotFound)
			return
		}
		targetID, targetName, targetMT = id, info.name, info.mediaType
	}

	s.scanMu.Lock()
	if s.scanning {
		s.scanMu.Unlock()
		http.Error(w, "Scan already in progress", http.StatusConflict)
		return
	}
	s.scanning = true
	s.scanMu.Unlock()

	scope := "all libraries"
	if libParam != "" {
		scope = fmt.Sprintf("library %s (ID: %s)", targetName, targetID)
	}
	fmt.Printf("[INFO] Manual scan triggered via /scan from %s: %s\n", r.RemoteAddr, scope)

	go func() {
		defer func() {
			s.scanMu.Lock()
			s.scanning = false
			s.scanMu.Unlock()
			fmt.Println("[INFO] Manual scan complete")
		}()
		if libParam == "" {
			s.scanner.RunAll()
			return
		}
		if err := s.scanner.RunLibrary(targetID, targetName, targetMT); err != nil {
			fmt.Printf("[ERROR] Manual scan of %s failed: %v\n", targetName, err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "scan started: %s\n", scope)
}

func (s *Server) findLibrary(param string) (string, libraryInfo, bool) {
	if info, ok := s.libraryMap[param]; ok {
		return param, info, true
	}
	lower := strings.ToLower(param)
	for id, info := range s.libraryMap {
		if strings.ToLower(info.name) == lower {
			return id, info, true
		}
	}
	return "", libraryInfo{}, false
}

func (s *Server) resolveMediaType(libraryID, sectionType string) media.MediaType {
	switch sectionType {
	case "movie":
		return media.MediaTypeMovie
	case "show":
		return media.MediaTypeTV
	}
	if info, ok := s.libraryMap[libraryID]; ok {
		return info.mediaType
	}
	return media.MediaTypeUnknown
}

// addPendingItem accumulates rating keys during the debounce window. Each new
// event for the same library resets the timer and adds the rating key to the list.
// When the timer fires, all accumulated keys are processed.
func (s *Server) addPendingItem(libraryID, libraryName string, mediaType media.MediaType, ratingKey string) {
	debounce := s.config.WebhookDebounce

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	pw, exists := s.pending[libraryID]
	if exists {
		pw.timer.Stop()
		pw.gen++
		if ratingKey != "" {
			pw.ratingKeys[ratingKey] = struct{}{}
		}
		if s.config.VerboseLogging {
			fmt.Printf("[WEBHOOK] reset debounce for library %s (%d items queued)\n", libraryName, len(pw.ratingKeys))
		}
	} else {
		keys := make(map[string]struct{})
		if ratingKey != "" {
			keys[ratingKey] = struct{}{}
		}
		pw = &pendingWork{
			libraryName: libraryName,
			mediaType:   mediaType,
			ratingKeys:  keys,
		}
		s.pending[libraryID] = pw
		fmt.Printf("[WEBHOOK] scheduled processing for library %s in %v\n", libraryName, debounce)
	}

	gen := pw.gen
	pw.timer = time.AfterFunc(debounce, func() {
		s.pendingMu.Lock()
		current, ok := s.pending[libraryID]
		if !ok || current.gen != gen {
			s.pendingMu.Unlock()
			return
		}
		keys := make([]string, 0, len(current.ratingKeys))
		for k := range current.ratingKeys {
			keys = append(keys, k)
		}
		delete(s.pending, libraryID)
		s.pendingMu.Unlock()

		s.processItems(libraryID, libraryName, mediaType, keys)
	})
}

func (s *Server) processItems(libraryID, libraryName string, mediaType media.MediaType, ratingKeys []string) {
	if len(ratingKeys) == 0 {
		fmt.Printf("[WEBHOOK] processing full library %s (no rating keys in events)\n", libraryName)
		if err := s.processor.ProcessAllItems(libraryID, libraryName, mediaType); err != nil {
			fmt.Printf("[WEBHOOK] error processing library %s: %v\n", libraryName, err)
		} else {
			fmt.Printf("[WEBHOOK] finished processing library %s\n", libraryName)
		}
		return
	}

	fmt.Printf("[WEBHOOK] processing %d items in library %s\n", len(ratingKeys), libraryName)
	for _, key := range ratingKeys {
		if err := s.processor.ProcessSingleItem(key, libraryID, mediaType); err != nil {
			fmt.Printf("[WEBHOOK] error processing item %s: %v\n", key, err)
		}
	}
	fmt.Printf("[WEBHOOK] finished processing %d items in library %s\n", len(ratingKeys), libraryName)
}
