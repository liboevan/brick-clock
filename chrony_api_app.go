package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	CHRONY_CONF_PATH = "/etc/chrony/chrony.conf"
	DEFAULT_SERVERS  = "pool.ntp.org"
)

// Response structures
type ServerResponse struct {
	Server string `json:"server"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

type SetServersRequest struct {
	Servers []string `json:"servers"`
}

type SetServerModeRequest struct {
	Enabled bool `json:"enabled"`
}

type StatusResponse struct {
	ServerModeEnabled bool                   `json:"server_mode_enabled"`
	Tracking          map[string]string      `json:"tracking"`
	TrackingError     string                 `json:"tracking_error"`
	Sources           []map[string]string    `json:"sources"`
	SourcesError      string                 `json:"sources_error"`
}

type VersionResponse struct {
	Version string `json:"version"`
	Error   string `json:"error"`
}

type ServerModeResponse struct {
	ServerModeEnabled bool `json:"server_mode_enabled"`
}

type SetServerModeResponse struct {
	Success           bool `json:"success"`
	ServerModeEnabled bool `json:"server_mode_enabled"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Helper function to run chronyc commands
func runChronyc(args []string) (string, string) {
	cmd := exec.Command("chronyc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err.Error()
	}
	return strings.TrimSpace(string(output)), ""
}

// Helper to read/write allow directive in chrony.conf
func getServerModeStatus() bool {
	content, err := ioutil.ReadFile(CHRONY_CONF_PATH)
	if err != nil {
		return false
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && strings.HasPrefix(strings.TrimSpace(line), "allow") {
			return true
		}
	}
	return false
}

func setServerModeStatus(enabled bool) bool {
	content, err := ioutil.ReadFile(CHRONY_CONF_PATH)
	if err != nil {
		return false
	}
	
	lines := strings.Split(string(content), "\n")
	var newLines []string
	found := false
	
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && strings.HasPrefix(strings.TrimSpace(line), "allow") {
			found = true
			if enabled {
				newLines = append(newLines, line)
			}
			// else: skip the line to disable
		} else {
			newLines = append(newLines, line)
		}
	}
	
	if enabled && !found {
		newLines = append(newLines, "allow 0.0.0.0/0")
	}
	
	newContent := strings.Join(newLines, "\n")
	err = ioutil.WriteFile(CHRONY_CONF_PATH, []byte(newContent), 0644)
	return err == nil
}

func parseSourcesOutput(output string) []map[string]string {
	lines := strings.Split(output, "\n")
	var sources []map[string]string
	headerFound := false
	
	for _, line := range lines {
		if !headerFound {
			if strings.TrimSpace(line) == "===============================================================================" {
				headerFound = true
			}
			continue
		}
		
		if strings.TrimSpace(line) == "" || strings.TrimSpace(line) == "===============================================================================" {
			continue
		}
		
		// Example line: ^? 198.18.5.209 0 7 0 - +0ns[   +0ns] +/- 0ns
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		if len(parts) >= 2 {
			sources = append(sources, map[string]string{
				"name": parts[1],
				"raw":  line,
			})
		}
	}
	return sources
}

func parseTrackingOutput(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return result
}

// API Handlers
func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	version, err := runChronyc([]string{"--version"})
	response := VersionResponse{
		Version: version,
		Error:   err,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	serverModeEnabled := getServerModeStatus()
	
	trackingRaw, trackingErr := runChronyc([]string{"tracking"})
	tracking := parseTrackingOutput(trackingRaw)
	
	sourcesRaw, sourcesErr := runChronyc([]string{"sources"})
	sources := parseSourcesOutput(sourcesRaw)
	
	response := StatusResponse{
		ServerModeEnabled: serverModeEnabled,
		Tracking:          tracking,
		TrackingError:     trackingErr,
		Sources:           sources,
		SourcesError:      sourcesErr,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		servers, err := runChronyc([]string{"sources"})
		response := map[string]string{
			"servers": servers,
			"error":   err,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		var req SetServersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		if len(req.Servers) == 0 {
			http.Error(w, "servers must be a non-empty list", http.StatusBadRequest)
			return
		}
		
		// Delete all current servers
		runChronyc([]string{"delete", "sources"})
		
		// Add new servers
		var responses []ServerResponse
		for _, server := range req.Servers {
			output, err := runChronyc([]string{"add", "server", server})
			responses = append(responses, ServerResponse{
				Server: server,
				Output: output,
				Error:  err,
			})
		}
		
		response := map[string]interface{}{
			"result": responses,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodDelete:
		output, err := runChronyc([]string{"delete", "sources"})
		response := map[string]string{
			"output": output,
			"error":  err,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleDefaultServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Delete all current servers
	runChronyc([]string{"delete", "sources"})
	
	// Add default server
	output, err := runChronyc([]string{"add", "server", DEFAULT_SERVERS})
	response := map[string]interface{}{
		"result": []ServerResponse{
			{
				Server: DEFAULT_SERVERS,
				Output: output,
				Error:  err,
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleServerMode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		enabled := getServerModeStatus()
		response := ServerModeResponse{
			ServerModeEnabled: enabled,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		var req SetServerModeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		success := setServerModeStatus(req.Enabled)
		response := SetServerModeResponse{
			Success:           success,
			ServerModeEnabled: req.Enabled,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	// Define routes
	http.HandleFunc("/chrony/version", handleVersion)
	http.HandleFunc("/chrony/status", handleStatus)
	http.HandleFunc("/chrony/servers", handleServers)
	http.HandleFunc("/chrony/servers/default", handleDefaultServers)
	http.HandleFunc("/chrony/server-mode", handleServerMode)
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	port := "8291"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	
	fmt.Printf("Starting Chrony API server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
} 