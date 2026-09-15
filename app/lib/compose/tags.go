package compose

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func UpdateImageTagEnv(envPath string, containerName string, imageTag string) error {
	// Append timestamped image tag update to tags.env file
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	envLine := fmt.Sprintf("# Updated %s\nIMAGE_TAG_%s=%s\n",
		timestamp,
		containerName,
		imageTag)
	var err error

	// Append to .env file (creates if doesn't exist)
	if _, err = os.Stat(envPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[ERROR] failed to write .env file: %v", err)
		return err
	}

	// If file didn't exist, this creates it. If it did, we need to append
	if os.IsNotExist(err) {
		if err := os.WriteFile(envPath, []byte(envLine), 0o644); err != nil {
			log.Printf("[ERROR] failed to create .env file: %v", err)
			return err
		}
	} else {
		// File exists, append to it
		f, err := os.OpenFile(envPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("[ERROR] failed to open .env for append: %v", err)
			return err
		}
		defer f.Close()

		if _, err := f.WriteString(envLine + "\n"); err != nil {
			log.Printf("[ERROR] failed to append to .env: %v", err)
			return err
		}
	}

	log.Printf("[INFO] Appended to .env: IMAGE_TAG_%s=%s at %s",
		containerName,
		imageTag,
		timestamp)
	return nil
}

type ImageTagHistory struct {
	Tag     string    `json:"Tag"`
	Created time.Time `json:"Created"`
}

func ListImageTags(envPath string) (map[string][]ImageTagHistory, error) {
	var err error
	if _, err = os.Stat(envPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[ERROR] failed to write .env file: %v", err)
		return nil, err
	}
	resultMap := make(map[string][]ImageTagHistory)
	if os.IsNotExist(err) {
		if err := os.WriteFile(envPath, []byte{}, 0o644); err != nil {
			log.Printf("[ERROR] failed to create .env file: %v", err)
			return make(map[string][]ImageTagHistory, 0), err
		}
	}
	data, err := os.ReadFile(envPath)
	if err != nil {
		log.Printf("[ERROR] failed to read .env file: %v", err)
		return nil, err
	}
	blocks := strings.Split(string(data), "\n\n")
	for _, tagBlock := range blocks {
		timeLine, tagLine, ok := strings.Cut(tagBlock, "\n")
		if !ok {
			continue
		}

		tagValuesIndex := 10
		if strings.HasPrefix(tagLine, "# IMAGE_TAG_") {
			tagValuesIndex = 12
		} else if strings.HasPrefix(tagLine, "IMAGE_TAG_") {
			tagValuesIndex = 10
		} else {
			continue
		}

		serviceName, tag, ok := strings.Cut(tagLine[tagValuesIndex:], "=")
		if !ok {
			continue
		}
		createdAt := strings.TrimPrefix(timeLine, "# Updated ")
		createdAt = strings.TrimSuffix(createdAt, " ")
		createdAtTime, err := time.Parse(time.DateTime, createdAt)
		if err != nil {
			continue
		}

		if _, ok := resultMap[serviceName]; !ok {
			resultMap[serviceName] = []ImageTagHistory{}
		}
		resultMap[serviceName] = append(resultMap[serviceName], ImageTagHistory{
			Tag:     tag,
			Created: createdAtTime,
		})
	}
	return resultMap, nil
}

func RollbackImageTag(envPath string, serviceName string, tag string) error {
	var err error
	if _, err = os.Stat(envPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[ERROR] failed to write .env file: %v", err)
		return err
	}
	data, err := os.ReadFile(envPath)
	if err != nil {
		log.Printf("[ERROR] failed to read .env file: %v", err)
		return err
	}
	buffer := bytes.NewBuffer([]byte{})

	blocks := strings.Split(string(data), "\n\n")
	tagFound := false
	for _, tagBlock := range blocks {
		timeLine, tagLine, ok := strings.Cut(tagBlock, "\n")
		if !ok {
			buffer.WriteString(tagBlock)
			buffer.WriteString("\n\n")
			continue
		}

		tagValuesIndex := 10
		if strings.HasPrefix(tagLine, "# IMAGE_TAG_") {
			tagValuesIndex = 12
		} else if strings.HasPrefix(tagLine, "IMAGE_TAG_") {
			tagValuesIndex = 10
		} else {
			buffer.WriteString(tagBlock)
			buffer.WriteString("\n\n")
			continue
		}

		serviceStr, tagStr, ok := strings.Cut(tagLine[tagValuesIndex:], "=")
		if !ok {
			buffer.WriteString(tagBlock)
			buffer.WriteString("\n\n")
			continue
		}
		if serviceName != serviceStr {
			buffer.WriteString(tagBlock)
			buffer.WriteString("\n\n")
			continue
		}
		if !tagFound {
			if tag != tagStr {
				buffer.WriteString(tagBlock)
				buffer.WriteString("\n\n")
				continue
			}
			tagFound = true
			if tagValuesIndex == 12 {
				// uncomment tag
				buffer.WriteString(timeLine)
				buffer.WriteString("\n")
				buffer.WriteString(tagLine[2:]) // remove "# " prefix
				buffer.WriteString("\n\n")
			} else {
				buffer.WriteString(tagBlock)
				buffer.WriteString("\n\n")
			}
			continue
		} else {
			if tag != tagStr {
				// comment tag
				buffer.WriteString(timeLine)
				buffer.WriteString("\n# ")
				buffer.WriteString(tagLine)
				buffer.WriteString("\n\n")

				continue
			}
		}
		buffer.WriteString(tagBlock)
		buffer.WriteString("\n\n")
	}
	if err := os.WriteFile(envPath, buffer.Bytes(), 0644); err != nil {
		log.Printf("[ERROR] failed to write .env file: %v", err)
		return err
	}
	return nil
}
