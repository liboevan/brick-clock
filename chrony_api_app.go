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
	BUILD_INFO_PATH  = "/build-info.json"
)

// Build info structure
type BuildInfo struct {
	Version        string `json:"version"`
	BuildDate      string `json:"buildDate"`
	BuildTimestamp int64  `json:"buildTimestamp"`
	Environment    string `json:"environment"`
	Service        string `json:"service"`
	Description    string `json:"description"`
}

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
	Activity          map[string]string      `json:"activity"`
	ActivityError     string                 `json:"activity_error"`
	Clients           []map[string]string    `json:"clients"`
	ClientsError      string                 `json:"clients_error"`
}

type VersionResponse struct {
	Version   string     `json:"version"`
	BuildInfo *BuildInfo `json:"buildInfo,omitempty"`
	Error     string     `json:"error"`
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

// Helper function to robustly restart chrony service in Alpine/docker environments
func restartChrony() bool {
	// Kill all running chronyd processes
	killCmd := exec.Command("pkill", "chronyd")
	_ = killCmd.Run() // Ignore error if not running

	// Start chronyd in the background
	startCmd := exec.Command("chronyd", "-f", CHRONY_CONF_PATH)
	err := startCmd.Start()
	if err != nil {
		log.Printf("Failed to start chronyd: %v", err)
		return false
	}
	log.Printf("chronyd restarted with PID %d", startCmd.Process.Pid)
	return true
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
	if err != nil {
		return false
	}
	
	// Restart chrony to apply the configuration changes
	return restartChrony()
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
		
		// Example line: ^* 202.118.1.130                 2   6   377    19   +625ms[ -117ms] +/-   25ms
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		if len(parts) >= 6 {
			source := map[string]string{
				"state":  parts[0],
				"name":   parts[1],
				"stratum": parts[2],
				"poll":   parts[3],
				"reach":  parts[4],
				"lastrx": parts[5],
				"raw":    line,
			}
			
			// Parse offset if available (format: +625ms[ -117ms] +/-   25ms)
			if len(parts) >= 6 {
				offsetPart := parts[6]
				// Extract offset value (e.g., +625ms)
				offsetMatch := regexp.MustCompile(`^([+-]\d+ms)`).FindStringSubmatch(offsetPart)
				if len(offsetMatch) > 1 {
					source["offset"] = offsetMatch[1]
				}
				
				// Extract delay if available
				if len(parts) >= 8 {
					delayPart := parts[8]
					delayMatch := regexp.MustCompile(`^([+-]\d+ms)`).FindStringSubmatch(delayPart)
					if len(delayMatch) > 1 {
						source["delay"] = delayMatch[1]
					}
				}
			}
			
			sources = append(sources, source)
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
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Map the keys to match what the frontend expects
				switch key {
				case "Reference ID":
					result["ReferenceID"] = value
				case "Stratum":
					result["Stratum"] = value
				case "Ref time (UTC)":
					result["Ref time (UTC)"] = value
				case "System time":
					result["System time"] = value
				case "Last offset":
					result["Last offset"] = value
				case "RMS offset":
					result["RMS offset"] = value
				case "Frequency":
					result["Frequency"] = value
				case "Residual freq":
					result["Residual freq"] = value
				case "Skew":
					result["Skew"] = value
				case "Root delay":
					result["Root delay"] = value
				case "Root dispersion":
					result["Root dispersion"] = value
				case "Update interval":
					result["Update interval"] = value
					// Also set UpdateRate for backward compatibility
					result["UpdateRate"] = value
				case "Leap status":
					result["Leap status"] = value
					// Also set LeapStatus for backward compatibility
					result["LeapStatus"] = value
				default:
					result[key] = value
				}
			}
		}
	}
	return result
}

func parseActivityOutput(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "sources") {
			// Parse lines like "1 sources online", "0 sources offline", etc.
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				count := parts[0]
				status := parts[2]
				switch status {
				case "online":
					result["ok_count"] = count
				case "offline":
					result["failed_count"] = count
				case "doing burst (return to online)":
					result["bogus_count"] = count
				case "doing burst (return to offline)":
					result["timeout_count"] = count
				}
			}
		}
	}
	return result
}

func parseClientsOutput(output string) []map[string]string {
	lines := strings.Split(output, "\n")
	var clients []map[string]string
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
		
		// Parse client line (format may vary)
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		if len(parts) >= 2 {
			client := map[string]string{
				"address": parts[0],
				"raw":     line,
			}
			
			// Try to parse additional fields if available
			if len(parts) >= 3 {
				client["ntp_packets"] = parts[1]
			}
			if len(parts) >= 4 {
				client["ntp_dropped"] = parts[2]
			}
			if len(parts) >= 5 {
				client["offset"] = parts[3]
			}
			
			clients = append(clients, client)
		}
	}
	return clients
}

// Helper function to load build info
func loadBuildInfo() *BuildInfo {
	data, err := ioutil.ReadFile(BUILD_INFO_PATH)
	if err != nil {
		return nil
	}
	
	var buildInfo BuildInfo
	if err := json.Unmarshal(data, &buildInfo); err != nil {
		return nil
	}
	
	return &buildInfo
}

// Helper function to read version from VERSION file
// Version and build info (will be replaced during build)
var (
	AppVersion    = "0.2.0"
	BuildDateTime = "2025-07-05T10:37:44Z"
)

func getVersion() string {
	// Return compiled-in version
	return AppVersion
}

// API Handlers
func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	version, err := runChronyc([]string{"--version"})
	buildInfo := loadBuildInfo()
	
	response := VersionResponse{
		Version:   version,
		BuildInfo: buildInfo,
		Error:     err,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleAppVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	version := getVersion()
	
	response := map[string]interface{}{
		"version":        version,
		"build_datetime": BuildDateTime,
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
	
	activityRaw, activityErr := runChronyc([]string{"activity"})
	activity := parseActivityOutput(activityRaw)
	
	clientsRaw, clientsErr := runChronyc([]string{"clients"})
	clients := parseClientsOutput(clientsRaw)
	
	response := StatusResponse{
		ServerModeEnabled: serverModeEnabled,
		Tracking:          tracking,
		TrackingError:     trackingErr,
		Sources:           sources,
		SourcesError:      sourcesErr,
		Activity:          activity,
		ActivityError:     activityErr,
		Clients:           clients,
		ClientsError:      clientsErr,
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
		
		// Restart chrony to apply the configuration changes
		restartSuccess := restartChrony()
		
		response := map[string]interface{}{
			"result": responses,
			"restart_success": restartSuccess,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodDelete:
		output, err := runChronyc([]string{"delete", "sources"})
		
		// Restart chrony to apply the configuration changes
		restartSuccess := restartChrony()
		
		response := map[string]interface{}{
			"output": output,
			"error":  err,
			"restart_success": restartSuccess,
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
	
	// Restart chrony to apply the configuration changes
	restartSuccess := restartChrony()
	
	response := map[string]interface{}{
		"result": []ServerResponse{
			{
				Server: DEFAULT_SERVERS,
				Output: output,
				Error:  err,
			},
		},
		"restart_success": restartSuccess,
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
	
	// Application version endpoint
	http.HandleFunc("/version", handleAppVersion)
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	port := "17003"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	
	fmt.Printf("Starting Brick Clock API server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
} 