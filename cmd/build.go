/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

var FilesPath string
var OutFilePath string

type (
	RuleFile struct {
		Groups []RuleGroup `yaml:"groups"`
	}

	RuleGroup struct {
		Name  string `yaml:"name"`
		Rules []Rule `yaml:"rules"`
	}

	Rule struct {
		AlertName   string            `yaml:"alert"`
		Expression  string            `yaml:"expr"`
		TimeFor     string            `yaml:"for"`
		Labels      map[string]string `yaml:"labels"`
		Annotations map[string]string `yaml:"annotations"`
	}
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatalf("%v", err)
		} else {
			build(path)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.PersistentFlags().StringVarP(&FilesPath, "path", "f", "", "alert rules path")
	buildCmd.PersistentFlags().StringVarP(&OutFilePath, "out", "o", "rules.yaml", "output rules file path")
	buildCmd.MarkFlagRequired("path")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func build(path string) {
	if !strings.HasSuffix(OutFilePath, ".yaml") {
		log.Fatal("OutFilePath has to be an yaml extension file")
	}
	var files []string
	err := filepath.Walk(path, visitFiles(&files, "yaml"))
	if err != nil {
		log.Fatal(err)
	}

	wg := sync.WaitGroup{}
	m := sync.Mutex{}

	duplicateAlerts := make(map[string][]string)
	ruleFile := &RuleFile{}

	groups := make(map[string][]Rule)
	wg.Add(len(files))
	for _, file := range files {
		go loadRuleFile(&wg, file, func(loadedGroups []RuleGroup) {
			m.Lock()
			defer m.Unlock()

			for _, group := range loadedGroups {
				groups[group.Name] = append(groups[group.Name], group.Rules...)
			}
		})
	}

	wg.Wait()

	for name, rules := range groups {
		duplicateAlerts[name] = getDuplicateRules(rules)
		ruleGroup := &RuleGroup{Name: name}
		ruleGroup.Rules = rules
		ruleFile.Groups = append(ruleFile.Groups, *ruleGroup)
	}

	raiseDuplicateAlerts(duplicateAlerts)
	writeRuleFile(ruleFile)
}

func loadRuleFile(wg *sync.WaitGroup, file string, loadedGroups func(groups []RuleGroup)) {
	ruleFile := RuleFile{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &ruleFile)
	if err != nil {
		log.Fatal(err)
	}

	loadedGroups(ruleFile.Groups)
	wg.Done()
}

func getDuplicateRules(rules []Rule) []string {
	var duplicateRules []string
	dict := make(map[string]int)
	for i, rule := range rules {
		if _, ok := dict[rule.AlertName]; ok {
			duplicateRules = append(duplicateRules, rule.AlertName)
		} else {
			dict[rule.AlertName] = i
		}
	}

	return duplicateRules
}

func visitFiles(files *[]string, ext string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if filepath.Ext(path) == fmt.Sprintf(".%s", ext) {
			*files = append(*files, path)
		}

		return nil
	}
}

func raiseDuplicateAlerts(duplicateAlerts map[string][]string) {
	for group, dup := range duplicateAlerts {
		if len(dup) > 0 {
			log.Printf("found duplicate alert names: %s under group: %s", strings.Join(dup, ","), group)
		}
	}
}

func writeRuleFile(ruleFile *RuleFile) {
	data, err := yaml.Marshal(ruleFile)
	if err != nil {
		log.Fatalf("failed to serialize rulefile:%v", err)
	}

	ioutil.WriteFile(OutFilePath, data, 0644)
}
