package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Tnze/go-mc/nbt"
)

const (
	fenceName  = "fence"
	podzolName = "podzol"
)

type Schematic struct {
	Blocks   []Block `json:"blocks"`
	Size     [3]int  `json:"size"`
	Offset   [3]int  `json:"offset"`
	Metadata struct {
		Format string `json:"format"`
	} `json:"metadata"`
}

type Block struct {
	Position [3]int  `json:"position"`
	Name     string  `json:"name"`
	Data     uint16  `json:"data,omitempty"`
	NBT      *string `json:"nbt,omitempty"`
}

func loadBlockMapping() (map[int]string, error) {
	data, err := ioutil.ReadFile("block_mapping.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load block mapping: %v", err)
	}

	var mapping map[string]string
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse block mapping: %v", err)
	}

	result := make(map[int]string)
	for k, v := range mapping {
		var key int
		if _, err := fmt.Sscanf(k, "%d", &key); err != nil {
			continue
		}
		result[key] = v
	}

	return result, nil
}

func promptForPath(prompt string) string {
	fmt.Print(prompt)
	var path string
	fmt.Scanln(&path)
	return path
}

func ConvertSchematicToJSON() error {
	blockMapping, err := loadBlockMapping()
	if err != nil {
		return err
	}

	inputPath := promptForPath("Enter schematic file path: ")
	inputPath = filepath.Clean(inputPath)

	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid gzip file: %v", err)
	}
	defer gzipReader.Close()

	buffer, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var schematic struct {
		Blocks    []byte `nbt:"Blocks"`
		Data      []byte `nbt:"Data"`
		Width     int    `nbt:"Width"`
		Length    int    `nbt:"Length"`
		Height    int    `nbt:"Height"`
		WEOffsetX int    `nbt:"WEOffsetX"`
		WEOffsetY int    `nbt:"WEOffsetY"`
		WEOffsetZ int    `nbt:"WEOffsetZ"`
	}

	if err := nbt.Unmarshal(buffer, &schematic); err != nil {
		return fmt.Errorf("failed to parse NBT data: %v", err)
	}

	if len(schematic.Blocks) == 0 {
		return fmt.Errorf("invalid schematic file - no block data")
	}

	result := Schematic{
		Size:   [3]int{schematic.Width, schematic.Height, schematic.Length},
		Offset: [3]int{schematic.WEOffsetX, schematic.WEOffsetY, schematic.WEOffsetZ},
	}
	result.Metadata.Format = "schematic"

	blockIndex := 0
	for y := 0; y < schematic.Height; y++ {
		for z := 0; z < schematic.Length; z++ {
			for x := 0; x < schematic.Width; x++ {
				blockID := int(schematic.Blocks[blockIndex])
				blockName, ok := blockMapping[blockID]
				if !ok {
					blockName = fmt.Sprintf("unknown_%d", blockID)
				}

				if blockName != "air" {
					block := Block{
						Position: [3]int{
							x + schematic.WEOffsetX,
							y + schematic.WEOffsetY,
							z + schematic.WEOffsetZ,
						},
						Name: blockName,
						Data: uint16(schematic.Data[blockIndex]),
					}

					if blockIndex-188 <= 5 && blockIndex-188 >= 0 {
						block.Name = fenceName
						block.Data = uint16(blockIndex - 188)
					}
					if blockIndex == 3 && block.Data == 2 {
						block.Name = podzolName
					}

					result.Blocks = append(result.Blocks, block)
				}
				blockIndex++
			}
		}
	}

	outputPath := promptForPath("Enter JSON output path (default: schematic.json): ")
	if outputPath == "" {
		outputPath = "schematic.json"
	}
	outputPath = filepath.Clean(outputPath)

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON: %v", err)
	}

	if err := ioutil.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("Successfully exported schematic to %s\n", outputPath)
	return nil
}

func main() {
	if err := ConvertSchematicToJSON(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
