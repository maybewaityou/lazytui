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

package ports

import "github.com/maybewaityou/lazytui/internal/core/domain"

// Repository is the data-source port. The default implementation is the JSON
// store; it can be swapped for an external-system adapter (e.g. ssh/tmux)
// without touching core. All methods key by Name.
type Repository interface {
	List() ([]domain.Item, error)
	Create(name string) (domain.Item, error) // error if name already exists
	Update(name string, item domain.Item) error
	Delete(name string) error
	Rename(oldName, newName string) error // mv semantics; error if newName exists
}
