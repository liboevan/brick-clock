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
	"strconv"
	"strings"
	"sync"
	"time"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/golang-jwt/jwt/v4"
)

const (
	CHRONY_CONF_PATH = "/etc/chrony/chrony.conf"
	DEFAULT_SERVERS  = "pool.ntp.org"
	BUILD_INFO_PATH  = "/build-info.json"
	STATUS_TRACKING    = 1
	STATUS_SOURCES     = 2
	STATUS_ACTIVITY    = 4
	STATUS_CLIENTS     = 8
	STATUS_SERVER_MODE = 16
	STATUS_ALL         = STATUS_TRACKING | STATUS_SOURCES | STATUS_ACTIVITY | STATUS_CLIENTS | STATUS_SERVER_MODE
)

// Build info structure
type BuildInfo struct {
	Version        string `json:"version"`
	BuildDateTime  string `json:"buildDateTime"`
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
	Tracking      map[string]string   `json:"tracking"`
	TrackingError string              `json:"tracking_error"`
	Sources       []map[string]string `json:"sources"`
	SourcesError  string              `json:"sources_error"`
	Activity      map[string]string   `json:"activity"`
	ActivityError string              `json:"activity_error"`
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

// Cache structures for lazy loading
type CachedData struct {
	Data      interface{}
	Timestamp time.Time
	TTL       time.Duration
	mutex     sync.RWMutex
	fetchData func() interface{}
}

func (c *CachedData) Get() interface{} {
	c.mutex.RLock()
	if time.Since(c.Timestamp) < c.TTL {
		defer c.mutex.RUnlock()
		return c.Data
	}
	c.mutex.RUnlock()
	
	// Only refresh when actually requested and stale
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.Timestamp) < c.TTL {
		return c.Data
	}
	
	// Refresh data here
	c.Data = c.fetchData()
	c.Timestamp = time.Now()
	return c.Data
}

// Global cache instances
var (
	trackingCache  *CachedData
	sourcesCache   *CachedData
	activityCache  *CachedData
	serverModeCache *CachedData
	clientsCache   *CachedData
	cacheInitialized bool
	cacheMutex     sync.Mutex
)

// Initialize caches
func initializeCaches() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	
	if cacheInitialized {
		return
	}
	
	// Initialize tracking cache (30 second TTL)
	trackingCache = &CachedData{
		TTL: 30 * time.Second,
	}
	trackingCache.fetchData = func() interface{} {
		output, err := runChronyc([]string{"tracking"})
		if err != "" {
			return map[string]string{"error": err}
		}
		return parseTrackingOutput(output)
	}
	
	// Initialize sources cache (30 second TTL)
	sourcesCache = &CachedData{
		TTL: 30 * time.Second,
	}
	sourcesCache.fetchData = func() interface{} {
		output, err := runChronyc([]string{"sources"})
		if err != "" {
			return []map[string]string{}
		}
		return parseSourcesOutput(output)
	}
	
	// Initialize activity cache (30 second TTL)
	activityCache = &CachedData{
		TTL: 30 * time.Second,
	}
	activityCache.fetchData = func() interface{} {
		output, err := runChronyc([]string{"activity"})
		if err != "" {
			return map[string]string{"error": err}
		}
		return parseActivityOutput(output)
	}
	
	// Initialize server mode cache (5 second TTL)
	serverModeCache = &CachedData{
		TTL: 5 * time.Second,
	}
	serverModeCache.fetchData = func() interface{} {
		return getServerModeStatus()
	}
	
	// Initialize clients cache (30 second TTL)
	clientsCache = &CachedData{
		TTL: 30 * time.Second,
	}
	clientsCache.fetchData = func() interface{} {
		output, err := runChronyc([]string{"clients"})
		if err != "" {
			return []map[string]string{}
		}
		return parseClientsOutput(output)
	}
	
	cacheInitialized = true
}

// Invalidate all caches to force refresh
func invalidateCaches() {
	if !cacheInitialized {
		return
	}
	
	// Invalidate tracking cache
	if trackingCache != nil {
		trackingCache.mutex.Lock()
		trackingCache.Timestamp = time.Time{} // Force refresh
		trackingCache.mutex.Unlock()
	}
	
	// Invalidate sources cache
	if sourcesCache != nil {
		sourcesCache.mutex.Lock()
		sourcesCache.Timestamp = time.Time{} // Force refresh
		sourcesCache.mutex.Unlock()
	}
	
	// Invalidate activity cache
	if activityCache != nil {
		activityCache.mutex.Lock()
		activityCache.Timestamp = time.Time{} // Force refresh
		activityCache.mutex.Unlock()
	}
	
	// Invalidate server mode cache
	if serverModeCache != nil {
		serverModeCache.mutex.Lock()
		serverModeCache.Timestamp = time.Time{} // Force refresh
		serverModeCache.mutex.Unlock()
	}
	
	// Invalidate clients cache
	if clientsCache != nil {
		clientsCache.mutex.Lock()
		clientsCache.Timestamp = time.Time{} // Force refresh
		clientsCache.mutex.Unlock()
	}
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
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "allow 0.0.0.0/0") {
			found = true
			if enabled {
				// If the line is commented, uncomment it
				if strings.HasPrefix(trimmed, "#") {
					newLines = append(newLines, "allow 0.0.0.0/0")
				} else {
					newLines = append(newLines, line)
				}
			} else {
				// If the line is not commented, comment it out
				if !strings.HasPrefix(trimmed, "#") {
					newLines = append(newLines, "#allow 0.0.0.0/0")
				} else {
					newLines = append(newLines, line)
				}
			}
		} else {
			newLines = append(newLines, line)
		}
	}
	
	// If enabling and not found, add the allow line
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
		// Log error for debugging
		fmt.Printf("Error reading build-info.json: %v\n", err)
		return nil
	}
	
	var buildInfo BuildInfo
	if err := json.Unmarshal(data, &buildInfo); err != nil {
		// Log error for debugging
		fmt.Printf("Error parsing build-info.json: %v\n", err)
		fmt.Printf("Raw data: %s\n", string(data))
		return nil
	}
	
	return &buildInfo
}

// Helper function to read version from VERSION file
// Version and build info (will be replaced during build)
var (
	AppVersion    = "0.1.0-dev"
	BuildDateTime = "2025-07-05T10:00:00Z"
)

func getVersion() string {
	// Return compiled-in version
	return AppVersion
}

var publicKey *rsa.PublicKey

func loadPublicKey(path string) *rsa.PublicKey {
	pemData, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read public.pem: %v", err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		log.Fatalf("Failed to decode PEM block")
	}
	// Try PKCS1 first
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err == nil {
		return pub
	}
	// Try PKIX (most common for 'PUBLIC KEY')
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if rsaPub, ok := parsed.(*rsa.PublicKey); ok {
			return rsaPub
		}
		log.Fatalf("Public key is not RSA")
	}
	log.Fatalf("Failed to parse public key: %v", err)
	return nil
}

func getClaimsFromRequest(r *http.Request) (map[string]interface{}, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing or invalid Authorization header")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

// API Handlers
func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	version := getVersion()
	buildInfo := loadBuildInfo()
	
	response := VersionResponse{
		Version:   version,
		BuildInfo: buildInfo,
		Error:     "",
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

	initializeCaches()

	flags := STATUS_ALL
	if flagStr := r.URL.Query().Get("flags"); flagStr != "" {
		if parsed, err := strconv.Atoi(flagStr); err == nil {
			flags = parsed
		}
	}

	response := make(map[string]interface{})

	if flags&STATUS_TRACKING != 0 {
		trackingData := trackingCache.Get()
		tracking, ok := trackingData.(map[string]string)
		if !ok {
			tracking = map[string]string{"error": "Failed to parse tracking data"}
		}
		response["tracking"] = tracking
	}

	if flags&STATUS_SOURCES != 0 {
		sourcesData := sourcesCache.Get()
		sources, ok := sourcesData.([]map[string]string)
		if !ok {
			sources = []map[string]string{}
		}
		response["sources"] = sources
	}

	if flags&STATUS_ACTIVITY != 0 {
		activityData := activityCache.Get()
		activity, ok := activityData.(map[string]string)
		if !ok {
			activity = map[string]string{"error": "Failed to parse activity data"}
		}
		response["activity"] = activity
	}

	if flags&STATUS_CLIENTS != 0 {
		clientsData := clientsCache.Get()
		clients, ok := clientsData.([]map[string]string)
		if !ok {
			clients = []map[string]string{}
		}
		response["clients"] = clients
	}

	if flags&STATUS_SERVER_MODE != 0 {
		serverModeData := serverModeCache.Get()
		enabled, ok := serverModeData.(bool)
		if !ok {
			enabled = false
		}
		response["server_mode_enabled"] = enabled
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Initialize caches if not already done
	initializeCaches()
	
	// Get tracking data from cache
	trackingData := trackingCache.Get()
	tracking, ok := trackingData.(map[string]string)
	if !ok {
		tracking = map[string]string{"error": "Failed to parse tracking data"}
	}
	
	response := map[string]interface{}{
		"tracking": tracking,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Initialize caches if not already done
	initializeCaches()
	
	// Get sources data from cache
	sourcesData := sourcesCache.Get()
	sources, ok := sourcesData.([]map[string]string)
	if !ok {
		sources = []map[string]string{}
	}
	
	response := map[string]interface{}{
		"sources": sources,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Initialize caches if not already done
	initializeCaches()
	
	// Get activity data from cache
	activityData := activityCache.Get()
	activity, ok := activityData.(map[string]string)
	if !ok {
		activity = map[string]string{"error": "Failed to parse activity data"}
	}
	
	response := map[string]interface{}{
		"activity": activity,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Initialize caches if not already done
	initializeCaches()
	
	// Get clients data from cache
	clientsData := clientsCache.Get()
	clients, ok := clientsData.([]map[string]string)
	if !ok {
		clients = []map[string]string{}
	}
	
	response := map[string]interface{}{
		"clients": clients,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper to update server list in chrony.conf
func updateChronyConfServers(servers []string) error {
	content, err := ioutil.ReadFile(CHRONY_CONF_PATH)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "server ") {
			newLines = append(newLines, line)
		}
	}
	// Add new server lines
	for _, server := range servers {
		newLines = append(newLines, "server "+server+" iburst")
	}
	// Ensure there's an empty line at the end
	newLines = append(newLines, "")
	newContent := strings.Join(newLines, "\n")
	return ioutil.WriteFile(CHRONY_CONF_PATH, []byte(newContent), 0644)
}

// Helper to read configured servers from chrony.conf
func getConfiguredServers() []string {
	content, err := ioutil.ReadFile(CHRONY_CONF_PATH)
	if err != nil {
		return []string{}
	}
	
	lines := strings.Split(string(content), "\n")
	var servers []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "server ") {
			// Extract server name from "server pool.ntp.org iburst"
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}
	return servers
}

func handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, err := getClaimsFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		// Return configured servers from chrony.conf, not active sources
		configuredServers := getConfiguredServers()
		response := map[string]interface{}{
			"servers": configuredServers,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		claims, err := getClaimsFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		if permissionCheckEnabled && !hasPermission(claims, "clock/servers") {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}
		var req SetServersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if len(req.Servers) == 0 {
			http.Error(w, "servers must be a non-empty list", http.StatusBadRequest)
			return
		}
		// Update chrony.conf with new servers
		err = updateChronyConfServers(req.Servers)
		if err != nil {
			http.Error(w, "Failed to update chrony.conf: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Restart chrony to apply the configuration changes
		restartSuccess := restartChrony()
		// Invalidate caches after configuration change
		invalidateCaches()
		response := map[string]interface{}{
			"result": req.Servers,
			"restart_success": restartSuccess,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodDelete:
		claims, err := getClaimsFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		if permissionCheckEnabled && !hasPermission(claims, "clock/servers") {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}
		output, errStr := runChronyc([]string{"delete", "sources"})
		// Restart chrony to apply the configuration changes
		restartSuccess := restartChrony()
		// Invalidate caches after configuration change
		invalidateCaches()
		response := map[string]interface{}{
			"output": output,
			"error":  errStr,
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

	// Persist default server to chrony.conf
	err := updateChronyConfServers([]string{DEFAULT_SERVERS})
	if err != nil {
		http.Error(w, "Failed to update chrony.conf: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Restart chrony to apply the configuration changes
	restartSuccess := restartChrony()

	// Invalidate caches after configuration change
	invalidateCaches()

	response := map[string]interface{}{
		"result": []string{DEFAULT_SERVERS},
		"restart_success": restartSuccess,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleServerMode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, err := getClaimsFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		// No permission check for GET
		// Initialize caches if not already done
		initializeCaches()
		
		// Get server mode from cache
		serverModeData := serverModeCache.Get()
		enabled, ok := serverModeData.(bool)
		if !ok {
			enabled = false
		}
		
		response := ServerModeResponse{
			ServerModeEnabled: enabled,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		claims, err := getClaimsFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		if permissionCheckEnabled && !hasPermission(claims, "clock/server_mode") {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}
		var req SetServerModeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		success := setServerModeStatus(req.Enabled)
		
		// Invalidate server mode cache after change
		if cacheInitialized && serverModeCache != nil {
			serverModeCache.mutex.Lock()
			serverModeCache.Timestamp = time.Time{} // Force refresh
			serverModeCache.mutex.Unlock()
		}
		
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

var permissionCheckEnabled = true

func init() {
	if os.Getenv("PERMISSION_CHECK") == "off" {
		permissionCheckEnabled = false
		log.Println("WARNING: Permission checks are DISABLED! Only authentication is enforced.")
	} else {
		permissionCheckEnabled = true
	}
}

// Helper to check if a user has a permission in JWT claims
func hasPermission(claims map[string]interface{}, perm string) bool {
	perms, ok := claims["permissions"]
	if !ok {
		return false
	}
	// permissions can be []interface{} or []string
	switch v := perms.(type) {
	case []interface{}:
		for _, p := range v {
			if ps, ok := p.(string); ok && ps == perm {
				return true
			}
		}
	case []string:
		for _, ps := range v {
			if ps == perm {
				return true
			}
		}
	case string:
		// fallback: comma-separated string
		for _, ps := range strings.Split(v, ",") {
			if strings.TrimSpace(ps) == perm {
				return true
			}
		}
	}
	return false
}

// Example usage in a handler (replace in all sensitive handlers):
//
// claims := getClaimsFromRequest(r) // your JWT parsing logic
// if permissionCheckEnabled && !hasPermission(claims, "clock/server-mode") {
//     http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
//     return
// }
// ...proceed with the action...

func main() {
	// Define routes - Hide chrony implementation details
	publicKey = loadPublicKey("/etc/brick/clock/public.pem") // adjust path as needed
	http.HandleFunc("/version", handleVersion)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/status/tracking", handleTracking)
	http.HandleFunc("/status/sources", handleSources)
	http.HandleFunc("/status/activity", handleActivity)
	http.HandleFunc("/status/clients", handleClients)
	http.HandleFunc("/servers", handleServers)
	http.HandleFunc("/servers/default", handleDefaultServers)
	http.HandleFunc("/server-mode", handleServerMode)
	
	// Application version endpoint
	http.HandleFunc("/app-version", handleAppVersion)
	
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