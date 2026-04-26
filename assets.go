package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0o755)
	}
	return nil
}

func (cfg apiConfig) getTypeExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func (cfg apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, key)
}

func (cfg apiConfig) getAssetPath(mediaType string) string {
	fileName := getFileName()
	ext := cfg.getTypeExt(mediaType)
	return fmt.Sprintf("%s%s", fileName, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func getFileName() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate random byte")
	}

	return base64.RawURLEncoding.EncodeToString(b)
}

func getVideoAspectRatio(filePath string) (string, error) {
	type ffprobeOuput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_streams",
		filePath)
	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return "", err
	}

	var output ffprobeOuput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		return "", err
	}

	if len(output.Streams) < 1 {
		return "", errors.New("ffprobe output stream is empty")
	}

	aspectRatio := aspectRatio(output.Streams[0].Width, output.Streams[0].Height)

	return aspectRatio, nil
}

func aspectRatio(w, h int) string {
	const (
		landscapeRatio = 16.0 / 9.0
		portraitRatio  = 9.0 / 16.0
		tolerance      = 0.02
	)
	videoRatio := float64(w) / float64(h)

	switch {
	case math.Abs(videoRatio-landscapeRatio) < tolerance:
		return "landscape"
	case math.Abs(videoRatio-portraitRatio) < tolerance:
		return "portrait"
	default:
		return "other"
	}
}
