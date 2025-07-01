package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppName    = "BloxDump"
	AppVersion = "v5.2.6-go"
)

// AssetType represents the type of asset found
type AssetType int

const (
	Unknown AssetType = iota
	Ignored
	NoConvert
	Mesh
	Khronos
	EXTM3U
	Translation
	FontList
	WebP
)

// ParsedCache represents a parsed cache entry
type ParsedCache struct {
	Success bool
	Link    string
	Content []byte
}

// Cache represents a cache entry (similar to Dumper.Cache in C#)
type Cache struct {
	Path string
	Data []byte
}

// ContentInfo holds information about identified content
type ContentInfo struct {
	Type      AssetType
	Extension string
	Format    string
	Category  string
}

var knownLinks = make(map[string]bool)

// Debug logging (simplified)
func debug(msg string) {
	fmt.Printf("\033[6;30;44mDEBUG\033[0m %s\n", msg)
}

func info(msg string) {
	fmt.Printf("\033[6;30;47mINFO\033[0m %s\n", msg)
}

func warn(msg string) {
	fmt.Printf("\033[6;30;43mWARN\033[0m %s\n", msg)
}

func errorMsg(msg string) {
	fmt.Printf("\033[6;30;41mERROR\033[0m %s\n", msg)
}

// IdentifyContent identifies the type of content based on byte signature
func IdentifyContent(content []byte) ContentInfo {
	if len(content) == 0 {
		return ContentInfo{Unknown, "", "", ""}
	}

	// Get the beginning of the file as string for pattern matching
	maxLen := 48
	if len(content) < maxLen {
		maxLen = len(content)
	}
	begin := string(content[:maxLen])

	// Check for OGG files specifically (what you need)
	if strings.HasPrefix(begin, "OggS") {
		return ContentInfo{NoConvert, "ogg", "OGG", "Sounds"}
	}

	// Other format checks (keeping some for completeness)
	if strings.Contains(begin, "<roblox!") {
		return ContentInfo{NoConvert, "rbxm", "RBXM", "RBXM"}
	}

	if strings.Contains(begin, "PNG\r\n") {
		return ContentInfo{NoConvert, "png", "PNG", "Textures"}
	}

	if strings.HasPrefix(begin, "GIF87a") || strings.HasPrefix(begin, "GIF89a") {
		return ContentInfo{NoConvert, "gif", "GIF", "Textures"}
	}

	if strings.Contains(begin, "JFIF") || strings.Contains(begin, "Exif") {
		return ContentInfo{NoConvert, "jfif", "JFIF", "Textures"}
	}

	// MP3 detection
	if strings.HasPrefix(begin, "ID3") || (len(content) > 2 && (content[0]&0xFF) == 0xFF && (content[1]&0xE0) == 0xE0) {
		return ContentInfo{NoConvert, "mp3", "MP3", "Sounds"}
	}

	// WebP detection
	if strings.HasPrefix(begin, "RIFF") && strings.Contains(begin, "WEBP") {
		return ContentInfo{NoConvert, "webp", "WebP", "Textures"}
	}

	return ContentInfo{Unknown, begin, "", ""}
}

// ParseCache parses a cache entry from a binary reader
func ParseCache(reader io.Reader) ParsedCache {
	buf := bufio.NewReader(reader)

	// Read magic header
	magic := make([]byte, 4)
	if _, err := io.ReadFull(buf, magic); err != nil {
		return ParsedCache{Success: false}
	}

	if string(magic) != "RBXH" {
		debug(fmt.Sprintf("Ignoring non-RBXH magic: %s", string(magic)))
		return ParsedCache{Success: false}
	}

	// Skip header size (4 bytes)
	if err := binary.Read(buf, binary.LittleEndian, &[]byte{0, 0, 0, 0}); err != nil {
		return ParsedCache{Success: false}
	}

	// Read link length
	var linkLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &linkLen); err != nil {
		return ParsedCache{Success: false}
	}

	// Read link
	linkBytes := make([]byte, linkLen)
	if _, err := io.ReadFull(buf, linkBytes); err != nil {
		return ParsedCache{Success: false}
	}
	link := string(linkBytes)

	// Check if we've already seen this link
	if knownLinks[link] {
		return ParsedCache{Success: false}
	}

	// Skip rogue byte
	buf.ReadByte()

	// Read status
	var status uint32
	if err := binary.Read(buf, binary.LittleEndian, &status); err != nil {
		return ParsedCache{Success: false}
	}

	if status >= 300 {
		return ParsedCache{Success: false}
	}

	// Read header length
	var headerLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &headerLen); err != nil {
		return ParsedCache{Success: false}
	}

	// Skip XXHash digest (4 bytes)
	buf.Discard(4)

	// Read content length
	var contentLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &contentLen); err != nil {
		return ParsedCache{Success: false}
	}

	// Skip XXHash digest, reserved bytes and headers
	buf.Discard(8 + int(headerLen))

	// Read content
	content := make([]byte, contentLen)
	if _, err := io.ReadFull(buf, content); err != nil {
		return ParsedCache{Success: false}
	}

	// Mark link as known
	knownLinks[link] = true

	return ParsedCache{
		Success: true,
		Link:    link,
		Content: content,
	}
}

// ParseCacheFromFile parses cache from a file path
func ParseCacheFromFile(cache Cache) ParsedCache {
	if cache.Data != nil {
		return ParseCache(bytes.NewReader(cache.Data))
	}

	if _, err := os.Stat(cache.Path); os.IsNotExist(err) {
		warn(fmt.Sprintf("Cache path not found: %s", cache.Path))
		return ParsedCache{Success: false}
	}

	file, err := os.Open(cache.Path)
	if err != nil {
		errorMsg(fmt.Sprintf("Error opening cache file: %v", err))
		return ParsedCache{Success: false}
	}
	defer file.Close()

	return ParseCache(file)
}

// ProcessOGGFile processes an OGG file and saves it
func ProcessOGGFile(content []byte, link string, outputDir string) error {
	// Create output directory if it doesn't exist
	soundsDir := filepath.Join(outputDir, "Sounds")
	if err := os.MkdirAll(soundsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sounds directory: %v", err)
	}

	// Generate filename from link or use a default
	filename := "sound.ogg"
	if link != "" {
		// Extract filename from URL or use hash
		parts := strings.Split(link, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if lastPart != "" {
				filename = lastPart
			}
		}
		// Ensure .ogg extension
		if !strings.HasSuffix(filename, ".ogg") {
			filename += ".ogg"
		}
	}

	outputPath := filepath.Join(soundsDir, filename)

	// Write the OGG file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write OGG file: %v", err)
	}

	info(fmt.Sprintf("Saved OGG file: %s", outputPath))
	return nil
}

// ProcessCacheFile processes a single cache file
func ProcessCacheFile(cachePath string, outputDir string) {
	cache := Cache{Path: cachePath}
	parsed := ParseCacheFromFile(cache)

	if !parsed.Success {
		return
	}

	contentInfo := IdentifyContent(parsed.Content)

	// Only process OGG files as requested
	if contentInfo.Type == NoConvert && contentInfo.Extension == "ogg" {
		info(fmt.Sprintf("Found OGG file: %s", parsed.Link))
		if err := ProcessOGGFile(parsed.Content, parsed.Link, outputDir); err != nil {
			errorMsg(fmt.Sprintf("Error processing OGG file: %v", err))
		}
	}
}

func main() {
	fmt.Printf("%s %s - OGG Extractor\n", AppName, AppVersion)

	if len(os.Args) < 3 {
		fmt.Println("Usage: bloxdump <cache_directory> <output_directory>")
		fmt.Println("Example: bloxdump C:\\Users\\User\\AppData\\Local\\Roblox\\http\\ ./output")
		os.Exit(1)
	}

	cacheDir := os.Args[1]
	outputDir := os.Args[2]

	info(fmt.Sprintf("Scanning cache directory: %s", cacheDir))
	info(fmt.Sprintf("Output directory: %s", outputDir))

	// Walk through all files in cache directory
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process each file as a potential cache file
		ProcessCacheFile(path, outputDir)
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking cache directory: %v", err)
	}

	info("Cache processing completed!")
}