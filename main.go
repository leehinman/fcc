package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	INT_ROOT string
	WARNINGS bool
)

const (
	PKG_DIR    = "packages"
	DS_DIR     = "data_stream"
	FIELDS_DIR = "fields"
)

type FieldDefinition struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"`
	External string            `yaml:"external"`
	Fields   []FieldDefinition `yaml:"fields"`
	Filename string
}

func init() {
	flag.StringVar(&INT_ROOT, "d", "./", "Location of integrations repo")
	flag.BoolVar(&WARNINGS, "w", false, "Turn on to see external warnings")
	flag.Parse()
}

func main() {
	fields := make(map[string]FieldDefinition)

	files, err := findFieldFiles(filepath.Join(INT_ROOT, PKG_DIR))
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		defs, err := fileDecode(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "file: %s, error: %v\n", file, err)
			continue
		}
		for _, def := range defs {
			def.Filename = file
			_, present := fields[def.Name]
			if !present {
				fields[def.Name] = def
				continue
			}
			if fields[def.Name].External != def.External {
				if !WARNINGS {
					continue
				}
				fmt.Printf("%s: mismatch in definitions\n", def.Name)
				fmt.Printf("\t%s external is %s\n", fields[def.Name].Filename, fields[def.Name].External)
				fmt.Printf("\t%s external is %s\n", def.Filename, def.External)
				continue
			}
			if fields[def.Name].Type != def.Type {
				fmt.Printf("%s: error in types\n", def.Name)
				fmt.Printf("\t%s type is %s\n", fields[def.Name].Filename, fields[def.Name].Type)
				fmt.Printf("\t%s type is %s\n", def.Filename, def.Type)
				continue
			}
		}
	}
}

func fileDecode(file string) ([]FieldDefinition, error) {
	defs := []FieldDefinition{}

	fp, err := os.Open(file)
	if err != nil {
		return defs, err
	}
	d := yaml.NewDecoder(fp)
	err = d.Decode(&defs)

	return flattenDefs(defs, ""), err
}

func flattenDefs(d []FieldDefinition, prefix string) []FieldDefinition {
	defs := []FieldDefinition{}
	for _, def := range d {
		if len(def.Fields) == 0 {
			if len(prefix) > 0 {
				def.Name = prefix + "." + def.Name
			}
			defs = append(defs, def)
			continue
		}
		if len(prefix) > 0 {
			def.Name = prefix + "." + def.Name
		}
		flat_defs := flattenDefs(def.Fields, def.Name)
		defs = append(defs, flat_defs...)
	}
	return defs
}

func findFieldFiles(root string) ([]string, error) {
	var found []string
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		components := strings.Split(path, string(os.PathSeparator))
		if len(components) < 4 {
			return nil
		}
		if components[len(components)-2] != FIELDS_DIR || components[len(components)-4] != DS_DIR {
			return nil
		}
		found = append(found, path)
		return nil
	})
	return found, err
}
