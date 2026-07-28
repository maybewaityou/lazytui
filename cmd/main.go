// Copyright 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/maybewaityou/lazytui/internal/adapters/data/store"
	"github.com/maybewaityou/lazytui/internal/adapters/ui"
	"github.com/maybewaityou/lazytui/internal/core/services"
	"github.com/maybewaityou/lazytui/internal/logger"
	"github.com/maybewaityou/lazytui/internal/tz"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func main() {
	log, err := logger.New("LAZYTUI")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	//nolint:errcheck // log.Sync error safe to ignore
	defer log.Sync()

	// Resolve the real local timezone before anything formats a time. Without
	// this, Termux/Android silently falls back to UTC (its zoneinfo lives under a
	// non-standard $PREFIX path Go's LoadLocation never searches), so every
	// .Local() timestamp renders hours off on a CST device. tz.Init embeds the
	// tzdb and rebinds time.Local from $TZ / Android getprop.
	if name := tz.Init(); name != "" {
		log.Infow("timezone resolved", "zone", name)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorw("home dir", "error", err)
		os.Exit(1)
	}
	dataPath := filepath.Join(home, ".lazytui", "items.json")

	repo, err := store.NewStore(dataPath)
	if err != nil {
		log.Errorw("store", "error", err)
		os.Exit(1)
	}
	svc := services.NewItemService(repo)
	t := ui.NewTUI(log, svc, version, gitCommit)

	root := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy TUI — generic list/details manager (template)",
		RunE:  func(*cobra.Command, []string) error { return t.Run() },
	}
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
