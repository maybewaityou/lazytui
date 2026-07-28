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

// Service is the business-logic port.
type Service interface {
	List() ([]domain.Item, error)
	Create(name string) error
	Update(name string, item domain.Item) error
	Delete(name string) error
	Rename(oldName, newName string) error
	TogglePin(name string) error
	SaveTags(name string, tags []string) error
	SaveNote(name string, note string) error
}

// ErrNotFound is the sentinel returned by Service methods that read-modify-write
// a single Item (TogglePin / SaveTags / SaveNote) when no Item with the given
// Name exists. It is a distinct type so callers can errors.Is against it without
// matching on a fragile string.
var ErrNotFound = errSentinel("item not found")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
