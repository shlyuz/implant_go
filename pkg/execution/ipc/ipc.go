package ipc

import (
	"bytes"
	"fmt" // For error wrapping
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// CreateNamedPipe creates a new named pipe (FIFO).
// It returns the path to the pipe or an error if creation fails.
func CreateNamedPipe() (string, error) {
	tmpDir, err := ioutil.TempDir("", "shlyuz-ipc-") // More specific prefix
	if err != nil {
		log.Printf("Error creating temp directory for named pipe: %v", err)
		return "", err
	}
	// It's good practice to use a more unique name for the pipe itself if multiple instances might run.
	// For now, "stdout" is kept as per original, but consider making it more unique if needed.
	namedPipe := filepath.Join(tmpDir, "pipefile") // Changed from "stdout" to "pipefile" for clarity

	if err := syscall.Mkfifo(namedPipe, 0600); err != nil {
		log.Printf("Error creating named pipe '%s': %v", namedPipe, err)
		// Attempt to clean up the temp directory if pipe creation fails
		_ = os.RemoveAll(tmpDir) // Best effort cleanup
		return "", err
	}
	log.Printf("Created named pipe: %s", namedPipe)
	return namedPipe, nil
}

// Read reads all data from the specified named pipe.
// It returns the data as a string or an error if any operation fails.
func Read(namedPipe string) (string, error) {
	log.Printf("Attempting to open named pipe for reading: %s", namedPipe)
	file, err := os.OpenFile(namedPipe, os.O_RDONLY, 0600)
	if err != nil {
		log.Printf("Error opening named pipe '%s' for reading: %v", namedPipe, err)
		return "", err
	}
	defer func() {
		log.Printf("Closing read pipe: %s", namedPipe)
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing read pipe '%s': %v", namedPipe, closeErr)
			// Decide if this error should supersede a previous error. Typically not.
		}
	}()

	var buff bytes.Buffer
	log.Printf("Reading from pipe: %s", namedPipe)
	if _, err := io.Copy(&buff, file); err != nil {
		log.Printf("Error reading from pipe '%s': %v", namedPipe, err)
		return "", err // buff will contain whatever was read before error
	}
	log.Printf("Finished reading from pipe: %s", namedPipe)
	return buff.String(), nil
}

// Write writes the given content to the specified named pipe.
// Returns an error if any operation fails.
func Write(namedPipe string, content []byte) error {
	log.Printf("Attempting to open named pipe for writing: %s", namedPipe)
	// Use O_WRONLY for writing. O_RDWR might be problematic if the other end isn't ready.
	file, err := os.OpenFile(namedPipe, os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Error opening named pipe '%s' for writing: %v", namedPipe, err)
		return err
	}
	defer func() {
		log.Printf("Closing write pipe: %s", namedPipe)
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing write pipe '%s': %v", namedPipe, closeErr)
		}
	}()

	log.Printf("Writing to pipe: %s", namedPipe)
	n, err := file.Write(content)
	if err != nil {
		log.Printf("Error writing to pipe '%s' (wrote %d bytes): %v", namedPipe, n, err)
		return err
	}
	if n < len(content) {
		log.Printf("Partial write to pipe '%s': wrote %d bytes, expected %d", namedPipe, n, len(content))
		return fmt.Errorf("partial write to pipe '%s': wrote %d, expected %d", namedPipe, n, len(content))
	}
	log.Printf("Finished writing to pipe: %s", namedPipe)
	return nil
}
