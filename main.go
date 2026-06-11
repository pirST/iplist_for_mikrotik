package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

const (
	defaultUpstream = "https://iplist.opencck.org/"
	defaultAddr     = ":8090"
)

var cidrPattern = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}/\d{1,2}\b`)

func main() {
	upstream := envOr("UPSTREAM_URL", defaultUpstream)
	addr := envOr("LISTEN_ADDR", defaultAddr)

	http.HandleFunc("/", proxyHandler(upstream))

	log.Printf("listening on %s, upstream: %s", addr, upstream)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func proxyHandler(upstream string) http.HandlerFunc {
	client := &http.Client{Timeout: 5 * time.Minute}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		query := r.URL.RawQuery
		clientIP := r.RemoteAddr

		target, err := buildUpstreamURL(upstream, query)
		if err != nil {
			logRequest(start, clientIP, query, http.StatusBadRequest, 0, 0, 0, 0, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			logRequest(start, clientIP, query, http.StatusInternalServerError, 0, 0, 0, 0, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			logRequest(start, clientIP, query, http.StatusBadGateway, 0, 0, 0, 0, err)
			http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logRequest(start, clientIP, query, http.StatusBadGateway, 0, 0, 0, len(body), err)
			http.Error(w, fmt.Sprintf("read upstream response: %v", err), http.StatusBadGateway)
			return
		}

		if resp.StatusCode != http.StatusOK {
			logRequest(start, clientIP, query, resp.StatusCode, 0, 0, 0, len(body), nil)
			for k, vv := range resp.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(body)
			return
		}

		result := deduplicateSubnets(body)
		logRequest(start, clientIP, query, http.StatusOK, result.linesIn, result.linesOut, result.removed, len(result.data), nil)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=iplist.rsc")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result.data)
	}
}

func buildUpstreamURL(upstream, rawQuery string) (string, error) {
	base, err := url.Parse(upstream)
	if err != nil {
		return "", fmt.Errorf("invalid upstream URL: %w", err)
	}
	base.RawQuery = rawQuery
	return base.String(), nil
}

type dedupResult struct {
	data     []byte
	linesIn  int
	linesOut int
	removed  int
}

func deduplicateSubnets(data []byte) dedupResult {
	seen := make(map[string]struct{})
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	linesIn := 0
	linesOut := 0
	removed := 0

	for scanner.Scan() {
		linesIn++
		line := scanner.Text()
		cidr := extractCIDR(line)
		if cidr != "" {
			if _, exists := seen[cidr]; exists {
				removed++
				continue
			}
			seen[cidr] = struct{}{}
		}
		linesOut++
		out.WriteString(line)
		out.WriteByte('\n')
	}

	return dedupResult{
		data:     out.Bytes(),
		linesIn:  linesIn,
		linesOut: linesOut,
		removed:  removed,
	}
}

func logRequest(start time.Time, clientIP, query string, status, linesIn, linesOut, removed, bytesOut int, err error) {
	duration := time.Since(start)

	if query == "" {
		query = "-"
	}

	if err != nil {
		log.Printf("[%s] %s | %d | %s | error: %v", clientIP, query, status, duration.Round(time.Millisecond), err)
		return
	}

	if linesIn > 0 {
		log.Printf("[%s] %s | %d | %s | lines: %d -> %d (removed %d) | %d bytes",
			clientIP, query, status, duration.Round(time.Millisecond), linesIn, linesOut, removed, bytesOut)
		return
	}

	log.Printf("[%s] %s | %d | %s | %d bytes", clientIP, query, status, duration.Round(time.Millisecond), bytesOut)
}

func extractCIDR(line string) string {
	match := cidrPattern.FindString(line)
	return match
}
