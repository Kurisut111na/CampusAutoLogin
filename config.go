package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// =============================================================================
// Windows DPAPI wrapper
// =============================================================================

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func dpapiProtect(plaintext []byte) ([]byte, error) {
	var inBlob dataBlob
	inBlob.cbData = uint32(len(plaintext))
	if len(plaintext) > 0 {
		inBlob.pbData = &plaintext[0]
	}

	var outBlob dataBlob
	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("CampusAutoLogin Password"))), // description
		0, // entropy (optional, nil = use user login)
		0, // reserved
		0, // prompt struct (nil)
		1, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	result := make([]byte, outBlob.cbData)
	copy(result, unsafe.Slice(outBlob.pbData, outBlob.cbData))
	return result, nil
}

func dpapiUnprotect(ciphertext []byte) ([]byte, error) {
	var inBlob dataBlob
	inBlob.cbData = uint32(len(ciphertext))
	if len(ciphertext) > 0 {
		inBlob.pbData = &ciphertext[0]
	}

	var outBlob dataBlob
	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // description out (optional)
		0, // entropy (nil = use user login)
		0, // reserved
		0, // prompt struct (nil)
		1, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	result := make([]byte, outBlob.cbData)
	copy(result, unsafe.Slice(outBlob.pbData, outBlob.cbData))
	return result, nil
}

// protectPassword encrypts a password string and returns base64 ciphertext.
func protectPassword(password string) (string, error) {
	encrypted, err := dpapiProtect([]byte(password))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// unprotectPassword decrypts a base64 ciphertext and returns the password string.
func unprotectPassword(base64Cipher string) (string, error) {
	cipher, err := base64.StdEncoding.DecodeString(base64Cipher)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	plain, err := dpapiUnprotect(cipher)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// =============================================================================
// AppConfig and ConfigManager
// =============================================================================

// AppConfig holds all application settings.
type AppConfig struct {
	Username          string   `json:"username"`
	Password          string   `json:"password"` // encrypted with DPAPI, base64-encoded
	Operator          string   `json:"operator"` // "cmcc", "unicom", "telecom"
	AutoLogin         bool     `json:"auto_login"`
	RememberPassword  bool     `json:"remember_password"`
	AutoStart         bool     `json:"auto_start"`
	StartMinimized    bool     `json:"start_minimized"`
	HeartbeatEnabled  bool     `json:"heartbeat_enabled"`
	HeartbeatInterval int      `json:"heartbeat_interval"` // seconds, 15-300
	CustomGateway     string   `json:"custom_gateway"`
	PingURLs          []string `json:"ping_urls"`
	LastLoginTime     string   `json:"last_login_time"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Operator:          "cmcc",
		RememberPassword:  true,
		HeartbeatEnabled:  true,
		HeartbeatInterval: 45,
		PingURLs:          []string{"https://www.baidu.com", "https://www.bing.com"},
	}
}

// ConfigManager handles loading and saving configuration.
type ConfigManager struct {
	mu         sync.Mutex
	configDir  string
	configFile string
}

// NewConfigManager creates a new ConfigManager.
func NewConfigManager() (*ConfigManager, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	configDir := filepath.Join(localAppData, "CampusAutoLogin")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return &ConfigManager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
	}, nil
}


// loadRaw reads the raw JSON config from disk (passwords still encrypted).
func (cm *ConfigManager) loadRaw() (*AppConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Corrupted config — start fresh
		GetLogger().Warn("Config file corrupted, starting with defaults")
		return DefaultConfig(), nil
	}
	return &cfg, nil
}

// LoadConfig reads config and decrypts the password.
func (cm *ConfigManager) LoadConfig() *AppConfig {
	cfg, err := cm.loadRaw()
	if err != nil {
		GetLogger().Error("Failed to load config: %v", err)
		return DefaultConfig()
	}

	// Decrypt password
	if cfg.Password != "" {
		plain, err := unprotectPassword(cfg.Password)
		if err != nil {
			GetLogger().Warn("Failed to decrypt password (different user/machine?): %v", err)
			// Don't clear the ciphertext — it might be valid on a different login
			// Just don't set the plain password
		} else {
			cfg.Password = plain
		}
	}
	return cfg
}

// SaveConfig encrypts the password and writes config to disk.
func (cm *ConfigManager) SaveConfig(cfg *AppConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Make a copy so we don't mutate the caller's config
	saveCfg := *cfg

	if !cfg.RememberPassword || cfg.Password == "" {
		saveCfg.Password = ""
	} else {
		// Check if password is already encrypted (base64 + DPAPI blob)
		// If it's plain text, encrypt it
		encrypted, err := protectPassword(cfg.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
		saveCfg.Password = encrypted
	}

	if saveCfg.HeartbeatInterval < 15 {
		saveCfg.HeartbeatInterval = 15
	}
	if saveCfg.HeartbeatInterval > 300 {
		saveCfg.HeartbeatInterval = 300
	}
	if len(saveCfg.PingURLs) == 0 {
		saveCfg.PingURLs = []string{"https://www.baidu.com", "https://www.bing.com"}
	}

	saveCfg.LastLoginTime = time.Now().Format("2006-01-02 15:04:05")

	data, err := json.MarshalIndent(saveCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configFile, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	GetLogger().Info("Config saved: %s", cm.configFile)
	return nil
}
